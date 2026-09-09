// macfit keeps Mac config files in one encrypted store, iCloud Drive by
// default, and restores them on any Mac. See README.md in this directory.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/queone/gkit/internal/lockbox"
	"golang.org/x/term"
)

const programVersion = "1.0.0"

// defaultStoreRel is the store location under the home directory when neither
// -s nor MACFIT_STORE names one: the root of iCloud Drive.
const defaultStoreRel = "Library/Mobile Documents/com~apple~CloudDocs/macfit.store"

// app carries the process-level dependencies so tests can swap them for fakes.
type app struct {
	stdout     io.Writer
	stderr     io.Writer
	keys       lockbox.KeyStore
	env        lockbox.Env
	host       string
	kdf        lockbox.KDF
	goos       string
	storeEnv   string
	isTerminal func() bool
	readSecret func(prompt string) ([]byte, error)
}

func newApp() *app {
	return &app{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		keys:       lockbox.SecurityKeyStore{},
		env:        lockbox.EnvFromOS(),
		host:       lockbox.Hostname(lockbox.DefaultExecutor, os.Hostname),
		kdf:        lockbox.DefaultKDF,
		goos:       runtime.GOOS,
		storeEnv:   os.Getenv("MACFIT_STORE"),
		isTerminal: func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		readSecret: terminalSecret,
	}
}

// terminalSecret prompts on stderr and reads one line with echo off.
func terminalSecret(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return b, err
}

func main() {
	os.Exit(newApp().run(os.Args[1:]))
}

func isVersionArg(a string) bool { return a == "-v" || a == "--version" || a == "v" || a == "version" }
func isHelpArg(a string) bool {
	return a == "-h" || a == "-?" || a == "--help" || a == "h" || a == "help"
}

func usage() string {
	return "macfit v" + programVersion + "\n" +
		"Keep Mac config files in one encrypted store and restore them on any Mac.\n\n" +
		"Overview\n" +
		"  The store is a single sealed file. Keep it in iCloud Drive, the default\n" +
		"  location, and every Mac on the account sees it; any other synced folder\n" +
		"  works the same way. push sends live files into the store, pull restores\n" +
		"  them from the store, and diff shows what differs. The key lives in the\n" +
		"  login keychain, with a passphrase-wrapped copy in the store for other Macs.\n\n" +
		"Usage\n" +
		"  macfit init                         create the store and key, or unlock an existing store\n" +
		"  macfit add PATH [-H HOST] [-l]      register a live file and capture it\n" +
		"  macfit rm TARGET [-H HOST]          forget a file and its stored versions\n" +
		"  macfit ls                           list entries\n" +
		"  macfit push [TARGET...]             send changed live files into the store\n" +
		"  macfit pull [TARGET...] [-n] [-f]   restore files from the store\n" +
		"  macfit diff [TARGET...] [-V]        show drift between the store and this Mac\n\n" +
		"Options\n" +
		"  -s, --store PATH   Store file (default: MACFIT_STORE, else macfit.store in iCloud Drive)\n" +
		"  -H, --host NAME    Bind the entry to one Mac (add, rm)\n" +
		"  -l, --literal      Keep the path under ~ instead of an XDG variable (add)\n" +
		"  -n, --dry-run      Print what pull would write and write nothing\n" +
		"  -f, --force        Let pull overwrite a live file that differs\n" +
		"  -V, --verbose      Add a unified diff to diff output\n" +
		"  -v, --version      Print macfit v" + programVersion + " and exit\n" +
		"  -h, -?, --help     Show this help message and exit\n\n" +
		"Notes\n" +
		"  A TARGET is the template ls shows ($XDG_CONFIG_HOME/git/config) or the live path.\n" +
		"  init needs a terminal for the passphrase prompt and never creates the iCloud Drive folder.\n" +
		"  Files only: no directories, globs, or symlinks. macOS defaults settings are a planned addition.\n"
}

func (a *app) errorf(format string, args ...any) {
	fmt.Fprintf(a.stderr, "macfit: "+format+"\n", args...)
}

// run dispatches macfit's subcommands and returns the process exit code.
func (a *app) run(args []string) int {
	if len(args) == 1 && isVersionArg(args[0]) {
		fmt.Fprintf(a.stdout, "macfit v%s\n", programVersion)
		return 0
	}
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		fmt.Fprint(a.stdout, usage())
		return 0
	}
	storePath, rest, err := a.splitStoreFlag(args)
	if err != nil {
		a.errorf("%s; run `macfit help`", err)
		return 2
	}
	if len(rest) == 0 || isHelpArg(rest[0]) {
		fmt.Fprint(a.stdout, usage())
		return 0
	}
	if a.goos != "darwin" {
		a.errorf("macfit supports macOS only")
		return 1
	}
	verb, vargs := rest[0], rest[1:]
	switch verb {
	case "init":
		return a.cmdInit(storePath, vargs)
	case "add":
		return a.cmdAdd(storePath, vargs)
	case "rm":
		return a.cmdRm(storePath, vargs)
	case "ls":
		return a.cmdLs(storePath, vargs)
	case "push":
		return a.cmdPush(storePath, vargs)
	case "pull":
		return a.cmdPull(storePath, vargs)
	case "diff":
		return a.cmdDiff(storePath, vargs)
	default:
		a.errorf("unknown command %q; run `macfit help`", verb)
		return 2
	}
}

// splitStoreFlag pulls -s/--store out of args wherever it appears and resolves
// the store path from the flag, the environment, or the default.
func (a *app) splitStoreFlag(args []string) (string, []string, error) {
	var path string
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-s" || arg == "--store":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("%s needs a PATH", arg)
			}
			path = args[i+1]
			i++
		case strings.HasPrefix(arg, "--store="):
			path = strings.TrimPrefix(arg, "--store=")
		default:
			rest = append(rest, arg)
		}
	}
	if path == "" {
		path = a.storeEnv
	}
	if path == "" {
		path = filepath.Join(a.env.Home, defaultStoreRel)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	return abs, rest, nil
}

type flagSpec struct {
	short string
	long  string
	value bool
}

// parseArgs separates known flags from positionals. A boolean flag maps to
// "true"; a value flag maps to its value, given as the next argument or
// after "=".
func parseArgs(args []string, specs []flagSpec) (map[string]string, []string, error) {
	flags := map[string]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			pos = append(pos, arg)
			continue
		}
		name, inline, hasInline := strings.Cut(arg, "=")
		var spec *flagSpec
		for k := range specs {
			if name == specs[k].short || name == specs[k].long {
				spec = &specs[k]
				break
			}
		}
		if spec == nil {
			return nil, nil, fmt.Errorf("unknown flag %s", name)
		}
		if !spec.value {
			if hasInline {
				return nil, nil, fmt.Errorf("%s takes no value", name)
			}
			flags[spec.long] = "true"
			continue
		}
		if hasInline {
			flags[spec.long] = inline
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("%s needs a value", name)
		}
		flags[spec.long] = args[i+1]
		i++
	}
	return flags, pos, nil
}

// openStore loads the store at path with the key from the keychain.
func (a *app) openStore(path string) (*lockbox.Store, error) {
	file, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no store at %s; run `macfit init`", path)
	}
	if err != nil {
		return nil, err
	}
	hdr, err := lockbox.ParseHeader(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	key, err := a.keys.Get(lockbox.KeyIDString(hdr.KeyID))
	if errors.Is(err, lockbox.ErrKeyNotFound) {
		return nil, fmt.Errorf("no key for %s in the login keychain; run `macfit init` and enter the recovery passphrase", path)
	}
	if err != nil {
		return nil, err
	}
	st, err := lockbox.Load(path, key)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if copies := lockbox.ConflictCopies(path); len(copies) > 0 {
		a.errorf("warning: iCloud conflict copies beside the store: %s", strings.Join(copies, ", "))
	}
	return st, nil
}

func (a *app) cmdInit(path string, args []string) int {
	if len(args) > 0 {
		a.errorf("init takes no arguments; run `macfit help`")
		return 2
	}
	file, err := os.ReadFile(path)
	switch {
	case err == nil:
		return a.initExisting(path, file)
	case errors.Is(err, fs.ErrNotExist):
		return a.initNew(path)
	default:
		a.errorf("init: read %s: %s", path, err)
		return 1
	}
}

// initExisting unwraps the key of an existing store with the recovery
// passphrase and saves it to this Mac's keychain.
func (a *app) initExisting(path string, file []byte) int {
	hdr, err := lockbox.ParseHeader(file)
	if err != nil {
		a.errorf("init: %s: %s", path, err)
		return 1
	}
	keyID := lockbox.KeyIDString(hdr.KeyID)
	_, err = a.keys.Get(keyID)
	if err == nil {
		a.errorf("init: store already initialized at %s and its key is in the login keychain", path)
		return 1
	}
	if !errors.Is(err, lockbox.ErrKeyNotFound) {
		a.errorf("init: %s", err)
		return 1
	}
	if !a.isTerminal() {
		a.errorf("init needs a terminal")
		return 1
	}
	pass, err := a.readSecret("Recovery passphrase for " + path + ": ")
	if err != nil {
		a.errorf("init: read passphrase: %s", err)
		return 1
	}
	key, err := hdr.UnwrapKey(pass)
	if err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	if err := a.keys.Put(keyID, key); err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "key for %s saved to the login keychain\n", path)
	return 0
}

// initNew creates a store, its key, and the passphrase-wrapped recovery copy.
func (a *app) initNew(path string) int {
	parent := filepath.Dir(path)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		a.errorf("init: directory %s does not exist; turn on iCloud Drive, or pass -s PATH to keep the store elsewhere", parent)
		return 1
	}
	if !a.isTerminal() {
		a.errorf("init needs a terminal")
		return 1
	}
	pass, err := a.readSecret("New recovery passphrase: ")
	if err != nil {
		a.errorf("init: read passphrase: %s", err)
		return 1
	}
	if len(bytes.TrimSpace(pass)) == 0 {
		a.errorf("init: passphrase must not be empty")
		return 1
	}
	again, err := a.readSecret("Repeat passphrase: ")
	if err != nil {
		a.errorf("init: read passphrase: %s", err)
		return 1
	}
	if !bytes.Equal(pass, again) {
		a.errorf("init: passphrases do not match")
		return 1
	}
	key, err := lockbox.NewKey()
	if err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	id, err := lockbox.NewKeyID()
	if err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	hdr := lockbox.Header{KeyID: id}
	if err := hdr.WrapKey(key, pass, a.kdf); err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	st, err := lockbox.Create(path, hdr, key)
	if err != nil {
		a.errorf("init: %s", err)
		return 1
	}
	defer st.Close()
	if err := st.Save(); err != nil {
		a.errorf("init: write %s: %s", path, err)
		return 1
	}
	if err := a.keys.Put(lockbox.KeyIDString(id), key); err != nil {
		a.errorf("init: store written but the key was not saved: %s; run `macfit init` again and enter the passphrase", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "store created at %s\nkey saved to the login keychain\nkeep the recovery passphrase safe: another Mac needs it once\n", path)
	return 0
}

func forHost(h string) string {
	if h == "" {
		return ""
	}
	return ", host " + h
}

func (a *app) cmdAdd(path string, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-H", "--host", true}, {"-l", "--literal", false}})
	if err != nil || len(pos) != 1 {
		a.errorf("add: usage: macfit add PATH [-H HOST] [-l]")
		return 2
	}
	abs, err := filepath.Abs(pos[0])
	if err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	info, err := os.Lstat(abs)
	if err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	if !info.Mode().IsRegular() {
		a.errorf("add: %s is not a regular file", abs)
		return 1
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	target := a.env.Template(abs)
	if flags["--literal"] == "true" {
		target = a.env.Literal(abs)
	}
	host := flags["--host"]
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	defer st.Close()
	e, err := st.AddEntry(target, info.Mode().Perm(), host)
	if errors.Is(err, lockbox.ErrExists) {
		a.errorf("add: %s is already registered%s; use `macfit push` to store its current content", target, forHost(host))
		return 1
	}
	if err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	if _, err := st.AddVersion(e.ID, content, a.host); err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	if err := st.Save(); err != nil {
		a.errorf("add: %s", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "added %s (mode %04o%s)\n", target, e.Mode, forHost(host))
	return 0
}

// targetKeys lists the store targets an argument may name: a template as
// given, or the templated, literal, and absolute forms of a live path.
func (a *app) targetKeys(arg string) []string {
	if strings.HasPrefix(arg, "~") || strings.HasPrefix(arg, "$") {
		return []string{arg}
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return []string{arg}
	}
	return []string{a.env.Template(abs), a.env.Literal(abs), abs}
}

func (a *app) cmdRm(path string, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-H", "--host", true}})
	if err != nil || len(pos) != 1 {
		a.errorf("rm: usage: macfit rm TARGET [-H HOST]")
		return 2
	}
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("rm: %s", err)
		return 1
	}
	defer st.Close()
	all, err := st.Entries()
	if err != nil {
		a.errorf("rm: %s", err)
		return 1
	}
	keys := a.targetKeys(pos[0])
	host, explicit := flags["--host"]
	var pick *lockbox.Entry
	for i := range all {
		e := &all[i]
		if !slices.Contains(keys, e.Target) {
			continue
		}
		switch {
		case explicit && e.Host == host:
			pick = e
		case !explicit && e.Host == a.host:
			pick = e
		case !explicit && e.Host == "" && pick == nil:
			pick = e
		}
	}
	if pick == nil {
		a.errorf("rm: %s is not registered%s", pos[0], forHost(host))
		return 1
	}
	if err := st.RemoveEntry(pick.ID); err != nil {
		a.errorf("rm: %s", err)
		return 1
	}
	if err := st.Save(); err != nil {
		a.errorf("rm: %s", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "removed %s%s\n", pick.Target, forHost(pick.Host))
	return 0
}

func (a *app) cmdLs(path string, args []string) int {
	if len(args) > 0 {
		a.errorf("ls takes no arguments; run `macfit help`")
		return 2
	}
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("ls: %s", err)
		return 1
	}
	defer st.Close()
	all, err := st.Entries()
	if err != nil {
		a.errorf("ls: %s", err)
		return 1
	}
	w := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TARGET\tMODE\tHOST\tCAPTURED")
	for _, e := range all {
		host := e.Host
		if host == "" {
			host = "-"
		}
		captured := "-"
		if v, ok, err := st.Latest(e.ID); err == nil && ok {
			captured = v.CapturedAt.Local().Format(time.DateTime)
		}
		fmt.Fprintf(w, "%s\t%04o\t%s\t%s\n", e.Target, e.Mode, host, captured)
	}
	w.Flush()
	return 0
}

// selected returns the entries that apply on this Mac, narrowed to args when given.
func (a *app) selected(st *lockbox.Store, args []string) ([]lockbox.Entry, error) {
	all, err := st.Entries()
	if err != nil {
		return nil, err
	}
	sel := lockbox.Select(all, a.host)
	if len(args) == 0 {
		return sel, nil
	}
	var out []lockbox.Entry
	for _, arg := range args {
		keys := a.targetKeys(arg)
		idx := slices.IndexFunc(sel, func(e lockbox.Entry) bool { return slices.Contains(keys, e.Target) })
		if idx < 0 {
			return nil, fmt.Errorf("%s is not registered for this Mac", arg)
		}
		out = append(out, sel[idx])
	}
	return out, nil
}

func (a *app) cmdPush(path string, args []string) int {
	_, pos, err := parseArgs(args, nil)
	if err != nil {
		a.errorf("push: %s; run `macfit help`", err)
		return 2
	}
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("push: %s", err)
		return 1
	}
	defer st.Close()
	sel, err := a.selected(st, pos)
	if err != nil {
		a.errorf("push: %s", err)
		return 1
	}
	rc, changed := 0, false
	for _, e := range sel {
		live := a.env.Expand(e.Target)
		content, err := os.ReadFile(live)
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(a.stdout, "missing    %s\n", e.Target)
			continue
		}
		if err != nil {
			a.errorf("push: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		info, err := os.Stat(live)
		if err != nil {
			a.errorf("push: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		mode := info.Mode().Perm()
		latest, ok, err := st.Latest(e.ID)
		if err != nil {
			a.errorf("push: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		sameContent := ok && latest.SHA256 == lockbox.Digest(content)
		if sameContent && mode == e.Mode {
			fmt.Fprintf(a.stdout, "unchanged  %s\n", e.Target)
			continue
		}
		if mode != e.Mode {
			if err := st.SetMode(e.ID, mode); err != nil {
				a.errorf("push: %s: %s", e.Target, err)
				return 1
			}
		}
		if !sameContent {
			if _, err := st.AddVersion(e.ID, content, a.host); err != nil {
				a.errorf("push: %s: %s", e.Target, err)
				return 1
			}
		}
		changed = true
		fmt.Fprintf(a.stdout, "updated    %s\n", e.Target)
	}
	if changed {
		if err := st.Save(); err != nil {
			a.errorf("push: %s", err)
			return 1
		}
	}
	return rc
}

func (a *app) cmdPull(path string, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-n", "--dry-run", false}, {"-f", "--force", false}})
	if err != nil {
		a.errorf("pull: %s; run `macfit help`", err)
		return 2
	}
	dry, force := flags["--dry-run"] == "true", flags["--force"] == "true"
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("pull: %s", err)
		return 1
	}
	defer st.Close()
	sel, err := a.selected(st, pos)
	if err != nil {
		a.errorf("pull: %s", err)
		return 1
	}
	rc := 0
	for _, e := range sel {
		latest, ok, err := st.Latest(e.ID)
		if err != nil {
			a.errorf("pull: %s: %s", e.Target, err)
			return 1
		}
		if !ok {
			fmt.Fprintf(a.stdout, "empty      %s (nothing stored yet)\n", e.Target)
			continue
		}
		live := a.env.Expand(e.Target)
		info, err := os.Lstat(live)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			fmt.Fprintf(a.stdout, "symlink    %s (refusing to write through a link)\n", e.Target)
			rc = 1
			continue
		}
		if err == nil {
			cur, rerr := os.ReadFile(live)
			if rerr != nil {
				a.errorf("pull: %s: %s", e.Target, rerr)
				rc = 1
				continue
			}
			if lockbox.Digest(cur) == latest.SHA256 && info.Mode().Perm() == e.Mode {
				fmt.Fprintf(a.stdout, "unchanged  %s\n", e.Target)
				continue
			}
			if !force {
				fmt.Fprintf(a.stdout, "differs    %s (use -f to overwrite)\n", e.Target)
				rc = 1
				continue
			}
		}
		if dry {
			fmt.Fprintf(a.stdout, "would write %s\n", e.Target)
			continue
		}
		dirMode := os.FileMode(0o755)
		if e.Mode == 0o600 {
			dirMode = 0o700
		}
		if err := os.MkdirAll(filepath.Dir(live), dirMode); err != nil {
			a.errorf("pull: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		if err := os.WriteFile(live, latest.Content, e.Mode); err != nil {
			a.errorf("pull: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		if err := os.Chmod(live, e.Mode); err != nil {
			a.errorf("pull: %s: %s", e.Target, err)
			rc = 1
			continue
		}
		fmt.Fprintf(a.stdout, "restored   %s\n", e.Target)
	}
	return rc
}

func (a *app) cmdDiff(path string, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-V", "--verbose", false}})
	if err != nil {
		a.errorf("diff: %s; run `macfit help`", err)
		return 2
	}
	verbose := flags["--verbose"] == "true"
	st, err := a.openStore(path)
	if err != nil {
		a.errorf("diff: %s", err)
		return 1
	}
	defer st.Close()
	sel, err := a.selected(st, pos)
	if err != nil {
		a.errorf("diff: %s", err)
		return 1
	}
	drift := false
	for _, e := range sel {
		latest, ok, err := st.Latest(e.ID)
		if err != nil {
			a.errorf("diff: %s: %s", e.Target, err)
			return 1
		}
		live := a.env.Expand(e.Target)
		cur, rerr := os.ReadFile(live)
		if errors.Is(rerr, fs.ErrNotExist) {
			fmt.Fprintf(a.stdout, "? %s\n", e.Target)
			drift = true
			continue
		}
		if rerr != nil {
			a.errorf("diff: %s: %s", e.Target, rerr)
			return 1
		}
		info, _ := os.Stat(live)
		mode := info.Mode().Perm()
		if !ok {
			fmt.Fprintf(a.stdout, "M %s (nothing stored yet)\n", e.Target)
			drift = true
			continue
		}
		sameContent := lockbox.Digest(cur) == latest.SHA256
		if sameContent && mode == e.Mode {
			fmt.Fprintf(a.stdout, "= %s\n", e.Target)
			continue
		}
		drift = true
		if mode != e.Mode {
			fmt.Fprintf(a.stdout, "M %s (mode %04o in store, %04o live)\n", e.Target, e.Mode, mode)
		} else {
			fmt.Fprintf(a.stdout, "M %s\n", e.Target)
		}
		if verbose && !sameContent {
			fmt.Fprint(a.stdout, unifiedDiff(e.Target+" (store)", live+" (live)", latest.Content, cur))
		}
	}
	if drift {
		return 1
	}
	return 0
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// unifiedDiff renders a single-hunk unified diff of a against b.
func unifiedDiff(aName, bName string, a, b []byte) string {
	al, bl := splitLines(a), splitLines(b)
	n, m := len(al), len(bl)
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", aName, bName)
	if n*m > 4_000_000 {
		sb.WriteString("(files differ; too large to diff line by line)\n")
		return sb.String()
	}
	// lcs[i][j] is the longest common subsequence length of al[i:] and bl[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	fmt.Fprintf(&sb, "@@ -1,%d +1,%d @@\n", n, m)
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && al[i] == bl[j]:
			sb.WriteString(" " + al[i] + "\n")
			i++
			j++
		case j < m && (i == n || lcs[i][j+1] > lcs[i+1][j]):
			sb.WriteString("+" + bl[j] + "\n")
			j++
		default:
			sb.WriteString("-" + al[i] + "\n")
			i++
		}
	}
	return sb.String()
}
