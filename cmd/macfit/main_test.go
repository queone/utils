package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/queone/gkit/internal/color"
	"github.com/queone/gkit/internal/lockbox"
)

// testKDF keeps passphrase derivation fast in tests.
var testKDF = lockbox.KDF{Time: 1, Memory: 8 * 1024, Threads: 1}

const yellow = "\x1b[38;5;220m"

type harness struct {
	t     *testing.T
	app   *app
	out   *bytes.Buffer
	errb  *bytes.Buffer
	root  string
	home  string
	store string
	keys  *lockbox.MemoryKeyStore
}

func testEnv(home string) lockbox.Env {
	return lockbox.Env{
		Home:       home,
		ConfigHome: filepath.Join(home, ".config"),
		DataHome:   filepath.Join(home, ".local", "share"),
		StateHome:  filepath.Join(home, ".local", "state"),
		CacheHome:  filepath.Join(home, ".cache"),
		ClaudeDir:  filepath.Join(home, ".config", "claude"),
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	synced := filepath.Join(root, "synced")
	if err := os.MkdirAll(synced, 0o755); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, out: &bytes.Buffer{}, errb: &bytes.Buffer{}, root: root, home: home,
		store: filepath.Join(synced, "macfit.store"), keys: &lockbox.MemoryKeyStore{}}
	h.app = &app{
		stdout: h.out, stderr: h.errb, keys: h.keys, env: testEnv(home), host: "a", kdf: testKDF, goos: "darwin",
		isTerminal: func() bool { return true },
		readSecret: func(string) ([]byte, error) { return []byte("pw"), nil },
		readLine:   func(string) (string, error) { return "", nil },
	}
	return h
}

// run executes macfit against the harness store through -s.
func (h *harness) run(args ...string) (int, string, string) {
	return h.runRaw(append([]string{"-s", h.store}, args...)...)
}

// runRaw executes macfit with the arguments as given, no -s added.
func (h *harness) runRaw(args ...string) (int, string, string) {
	h.out.Reset()
	h.errb.Reset()
	code := h.app.run(args)
	return code, h.out.String(), h.errb.String()
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	code, out, errs := h.run(args...)
	if code != 0 {
		h.t.Fatalf("macfit %s: exit %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out, errs)
	}
	return out
}

func (h *harness) mustFail(want int, args ...string) (string, string) {
	h.t.Helper()
	code, out, errs := h.run(args...)
	if code != want {
		h.t.Fatalf("macfit %s: exit %d, want %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, want, out, errs)
	}
	return out, errs
}

func (h *harness) write(rel, content string, mode os.FileMode) string {
	h.t.Helper()
	p := filepath.Join(h.home, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		h.t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		h.t.Fatal(err)
	}
	return p
}

func (h *harness) read(rel string) string {
	h.t.Helper()
	b, err := os.ReadFile(filepath.Join(h.home, rel))
	if err != nil {
		h.t.Fatal(err)
	}
	return string(b)
}

func (h *harness) mode(rel string) os.FileMode {
	h.t.Helper()
	info, err := os.Stat(filepath.Join(h.home, rel))
	if err != nil {
		h.t.Fatal(err)
	}
	return info.Mode().Perm()
}

func (h *harness) openStore() *lockbox.Store {
	h.t.Helper()
	st, err := h.app.openStore(h.store)
	if err != nil {
		h.t.Fatal(err)
	}
	return st
}

func (h *harness) generation() uint64 {
	h.t.Helper()
	b, err := os.ReadFile(h.store)
	if err != nil {
		h.t.Fatal(err)
	}
	hdr, err := lockbox.ParseHeader(b)
	if err != nil {
		h.t.Fatal(err)
	}
	return hdr.Generation
}

func (h *harness) pointer() string {
	h.t.Helper()
	b, err := os.ReadFile(h.app.pointerFile())
	if err != nil {
		return ""
	}
	return string(b)
}

func TestVersionAndHelp(t *testing.T) {
	h := newHarness(t)
	for _, arg := range []string{"--version", "-v", "v", "version"} {
		if code, out, errs := h.runRaw(arg); code != 0 || out != "macfit v1.1.0\n" || errs != "" {
			t.Fatalf("%s: code %d stdout %q stderr %q", arg, code, out, errs)
		}
	}
	if code, out, _ := h.runRaw(); code != 0 || !strings.Contains(out, "\nUsage\n") {
		t.Fatalf("bare invocation: code %d out %q", code, out)
	}
	if code, out, _ := h.runRaw("help"); code != 0 || !strings.Contains(out, "macfit key show") {
		t.Fatalf("help: code %d out %q", code, out)
	}
	if code, _, errs := h.run("bogus"); code != 2 || !strings.Contains(errs, "unknown command") {
		t.Fatalf("unknown command: code %d stderr %q", code, errs)
	}
	if code, _, errs := h.run("add", "--nope", "x"); code != 2 || !strings.Contains(errs, "usage") {
		t.Fatalf("unknown flag: code %d stderr %q", code, errs)
	}
}

func TestHelpLayoutMatchesTheOtherUtilities(t *testing.T) {
	h := newHarness(t)
	for _, arg := range []string{"help", "-h", "-?", "--help", "h"} {
		code, out, _ := h.runRaw(arg)
		if code != 0 {
			t.Fatalf("%s: code %d", arg, code)
		}
		lines := strings.Split(out, "\n")
		if lines[0] != "macfit v1.1.0" {
			t.Fatalf("%s: first line %q", arg, lines[0])
		}
		if lines[1] != "Keep Mac config files in one encrypted store and restore them on any Mac." {
			t.Fatalf("%s: second line %q", arg, lines[1])
		}
		last := -1
		for _, section := range []string{"\nOverview\n", "\nUsage\n", "\nOptions\n", "\nNotes\n"} {
			idx := strings.Index(out, section)
			if idx < 0 || idx < last {
				t.Fatalf("%s: section %q missing or out of order", arg, strings.TrimSpace(section))
			}
			last = idx
		}
		for _, want := range []string{"  -N, --new ", "  -h, -?, --help     Show this help message and exit", "Store path order: -s, then MACFIT_STORE"} {
			if !strings.Contains(out, want) {
				t.Fatalf("%s: help lacks %q", arg, want)
			}
		}
	}
}

func TestHelpHeaderAndHeadingsAreColoredLikeSkout(t *testing.T) {
	h := newHarness(t)
	plain := color.ClearCode(usage())
	defer color.SetEnabled(true)()
	_, out, _ := h.runRaw("help")
	lines := strings.Split(out, "\n")
	if lines[0] != color.Bold(color.Gra10("macfit"))+" v1.1.0" {
		t.Fatalf("first line %q", lines[0])
	}
	if lines[1] != color.Gra5("Keep Mac config files in one encrypted store and restore them on any Mac.") {
		t.Fatalf("second line %q", lines[1])
	}
	for _, name := range []string{"Overview", "Usage", "Options", "Notes"} {
		if !strings.Contains(out, "\n"+color.Bold(color.Gra10(name))+"\n") {
			t.Fatalf("heading %s is not bold white: %q", name, out)
		}
	}
	if color.ClearCode(out) != plain {
		t.Fatal("stripping escapes does not yield the plain help")
	}
	restore := color.SetEnabled(false)
	_, out, _ = h.runRaw("help")
	restore()
	if strings.Contains(out, "\x1b[") || out != plain {
		t.Fatalf("color disabled: %q", out)
	}
}

func TestReadmeUsageBlockEqualsHelp(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	start := strings.Index(readme, "### Usage\n\n```text\n")
	if start < 0 {
		t.Fatal("README has no ### Usage block")
	}
	start += len("### Usage\n\n```text\n")
	end := strings.Index(readme[start:], "```")
	if end < 0 {
		t.Fatal("README usage block is not closed")
	}
	if got, want := readme[start:start+end], color.ClearCode(usage()); got != want {
		t.Fatalf("README usage block differs from help\n--- README\n%s\n--- help\n%s", got, want)
	}
}

func TestPlatformGuard(t *testing.T) {
	h := newHarness(t)
	h.app.goos = "linux"
	for _, verb := range []string{"init", "add", "rm", "ls", "push", "pull", "diff", "key"} {
		code, _, errs := h.run(verb)
		if code != 1 || !strings.Contains(errs, "macfit supports macOS only") {
			t.Fatalf("%s on linux: code %d stderr %q", verb, code, errs)
		}
	}
	if code, _, _ := h.runRaw("--version"); code != 0 {
		t.Fatalf("--version on linux: code %d", code)
	}
	if code, _, _ := h.runRaw("help"); code != 0 {
		t.Fatalf("help on linux: code %d", code)
	}
}

func TestStoreResolutionOrder(t *testing.T) {
	h := newHarness(t)
	def := filepath.Join(h.home, ".local", "share", "macfit", "macfit.store")
	_, out, _ := h.runRaw("key", "show")
	if !strings.HasPrefix(out, "store: "+def+" (default)\n") {
		t.Fatalf("default: %q", out)
	}
	if err := os.MkdirAll(filepath.Dir(h.app.pointerFile()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.app.pointerFile(), []byte(filepath.Join(h.root, "pointed.store")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, out, _ = h.runRaw("key", "show")
	if !strings.HasPrefix(out, "store: "+filepath.Join(h.root, "pointed.store")+" (pointer)\n") {
		t.Fatalf("pointer: %q", out)
	}
	h.app.storeEnv = filepath.Join(h.root, "env.store")
	_, out, _ = h.runRaw("key", "show")
	if !strings.HasPrefix(out, "store: "+filepath.Join(h.root, "env.store")+" (env)\n") {
		t.Fatalf("env over pointer: %q", out)
	}
	_, out, _ = h.run("key", "show")
	if !strings.HasPrefix(out, "store: "+h.store+" (flag)\n") {
		t.Fatalf("flag over env: %q", out)
	}
}

func TestInitNewCreatesAndRefuses(t *testing.T) {
	h := newHarness(t)
	code, out, errs := h.runRaw("init", "-N")
	if code != 0 || !strings.Contains(out, "store created at ") {
		t.Fatalf("default init -N: code %d out %q err %q", code, out, errs)
	}
	dir := filepath.Join(h.home, ".local", "share", "macfit")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("default folder mode %o, want 700", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(dir, "macfit.store")); err != nil {
		t.Fatal("default store not created")
	}
	if h.pointer() != "" {
		t.Fatal("init -N at the default path must not write a pointer")
	}

	h.store = filepath.Join(h.root, "missing", "x.store")
	if _, errs := h.mustFail(1, "init", "-N"); !strings.Contains(errs, "does not exist") {
		t.Fatalf("missing parent: %q", errs)
	}
	if _, err := os.Stat(filepath.Join(h.root, "missing")); !os.IsNotExist(err) {
		t.Fatal("init -N created the missing parent")
	}

	h.store = filepath.Join(h.root, "synced", "macfit.store")
	out = h.mustRun("init", "-N")
	if !strings.Contains(out, "store path remembered in "+h.app.pointerFile()) {
		t.Fatalf("init -N -s: %q", out)
	}
	pinfo, err := os.Stat(h.app.pointerFile())
	if err != nil {
		t.Fatal(err)
	}
	if pinfo.Mode().Perm() != 0o600 || h.pointer() != h.store+"\n" {
		t.Fatalf("pointer mode %o content %q", pinfo.Mode().Perm(), h.pointer())
	}
	before, _ := os.ReadFile(h.store)
	if _, errs := h.mustFail(1, "init", "-N"); !strings.Contains(errs, "already exists") {
		t.Fatalf("init -N over existing: %q", errs)
	}
	after, _ := os.ReadFile(h.store)
	if !bytes.Equal(before, after) {
		t.Fatal("init -N over an existing store changed it")
	}
}

func TestInitUnlocksAndRemembers(t *testing.T) {
	h := newHarness(t)
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "init -N") {
		t.Fatalf("init without store: %q", errs)
	}
	h.mustRun("init", "-N")
	var savedID string
	var savedKey []byte
	for id, k := range h.keys.Keys {
		savedID, savedKey = id, k
	}

	other := newHarness(t)
	other.store = h.store
	out := other.mustRun("init")
	if !strings.Contains(out, "saved to the login keychain") || !strings.Contains(out, "store path remembered") {
		t.Fatalf("other Mac init: %q", out)
	}
	if got := other.keys.Keys[savedID]; !bytes.Equal(got, savedKey) {
		t.Fatal("other Mac recovered a different key")
	}
	if code, _, errs := other.runRaw("ls"); code != 0 {
		t.Fatalf("ls through the pointer: code %d stderr %q", code, errs)
	}
	if out := other.mustRun("init"); !strings.Contains(out, "already unlocked") {
		t.Fatalf("init on an unlocked store: %q", out)
	}

	wrong := newHarness(t)
	wrong.store = h.store
	wrong.app.readSecret = func(string) ([]byte, error) { return []byte("nope"), nil }
	if _, errs := wrong.mustFail(1, "init"); !strings.Contains(errs, "wrong passphrase") {
		t.Fatalf("wrong passphrase: %q", errs)
	}
	if len(wrong.keys.Keys) != 0 || wrong.pointer() != "" {
		t.Fatal("wrong passphrase must save neither key nor pointer")
	}
	if _, errs := wrong.mustFail(1, "ls"); !strings.Contains(errs, "no key for") {
		t.Fatalf("ls without key: %q", errs)
	}
}

func TestRelativeStoreFlagIsMadeAbsolute(t *testing.T) {
	h := newHarness(t)
	if err := os.MkdirAll(filepath.Join(h.root, "rel"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(h.root)
	if code, _, errs := h.runRaw("-s", "rel/macfit.store", "init", "-N"); code != 0 {
		t.Fatalf("relative -s: code %d stderr %q", code, errs)
	}
	want, _ := filepath.EvalSymlinks(filepath.Join(h.root, "rel"))
	got, _ := filepath.EvalSymlinks(filepath.Dir(strings.TrimSpace(h.pointer())))
	if got != want || !filepath.IsAbs(strings.TrimSpace(h.pointer())) {
		t.Fatalf("pointer holds %q, want a path under %q", h.pointer(), want)
	}
}

func TestInitRefusalsChangeNothing(t *testing.T) {
	h := newHarness(t)
	h.app.isTerminal = func() bool { return false }
	if _, errs := h.mustFail(1, "init", "-N"); !strings.Contains(errs, "init needs a terminal") {
		t.Fatalf("non-terminal: %q", errs)
	}
	h.app.isTerminal = func() bool { return true }
	answers := [][]byte{[]byte("one"), []byte("two")}
	h.app.readSecret = func(string) ([]byte, error) {
		a := answers[0]
		answers = answers[1:]
		return a, nil
	}
	if _, errs := h.mustFail(1, "init", "-N"); !strings.Contains(errs, "do not match") {
		t.Fatalf("mismatch: %q", errs)
	}
	h.app.readSecret = func(string) ([]byte, error) { return []byte("  "), nil }
	if _, errs := h.mustFail(1, "init", "-N"); !strings.Contains(errs, "must not be empty") {
		t.Fatalf("empty: %q", errs)
	}
	if _, err := os.Stat(h.store); !os.IsNotExist(err) {
		t.Fatal("a refused init wrote a store")
	}
	if h.pointer() != "" {
		t.Fatal("a refused init wrote a pointer")
	}
}

func TestAddPushDiffPullRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	rel := ".config/git/config"
	live := h.write(rel, "[user]\n\tname = a\n", 0o600)
	const target = "$XDG_CONFIG_HOME/git/config"

	out := h.mustRun("add", live)
	if !strings.Contains(out, "added "+target+" (mode 0600)") {
		t.Fatalf("add: %q", out)
	}
	out = h.mustRun("ls")
	if !strings.Contains(out, target) || !strings.Contains(out, "0600") {
		t.Fatalf("ls: %q", out)
	}
	if out := h.mustRun("diff"); out != "= "+target+"\n" {
		t.Fatalf("diff clean: %q", out)
	}
	if out := h.mustRun("push"); out != "unchanged  "+target+"\n" {
		t.Fatalf("push unchanged: %q", out)
	}

	h.write(rel, "[user]\n\tname = b\n", 0o600)
	out, _ = h.mustFail(1, "diff")
	if out != "M "+target+"\n" {
		t.Fatalf("diff modified: %q", out)
	}
	out, _ = h.mustFail(1, "diff", "-V")
	if !strings.Contains(out, "-\tname = a") || !strings.Contains(out, "+\tname = b") || !strings.Contains(out, "@@ -1,2 +1,2 @@") {
		t.Fatalf("diff -V: %q", out)
	}
	if out := h.mustRun("push"); out != "updated    "+target+"\n" {
		t.Fatalf("push updated: %q", out)
	}
	st := h.openStore()
	entries, _ := st.Entries()
	if n, _ := st.VersionCount(entries[0].ID); n != 2 {
		t.Fatalf("versions after second push: %d, want 2", n)
	}
	st.Close()
	if out := h.mustRun("push"); out != "unchanged  "+target+"\n" {
		t.Fatalf("push after update: %q", out)
	}

	if err := os.Remove(live); err != nil {
		t.Fatal(err)
	}
	if out, _ := h.mustFail(1, "diff"); out != "? "+target+"\n" {
		t.Fatalf("diff missing: %q", out)
	}
	if out := h.mustRun("push"); out != "missing    "+target+"\n" {
		t.Fatalf("push missing: %q", out)
	}
	if out := h.mustRun("pull"); out != "restored   "+target+"\n" {
		t.Fatalf("pull: %q", out)
	}
	if got := h.read(rel); got != "[user]\n\tname = b\n" {
		t.Fatalf("restored content %q", got)
	}
	if m := h.mode(rel); m != 0o600 {
		t.Fatalf("restored mode %o", m)
	}

	if err := os.RemoveAll(filepath.Join(h.home, ".config", "git")); err != nil {
		t.Fatal(err)
	}
	h.mustRun("pull")
	dirInfo, err := os.Stat(filepath.Join(h.home, ".config", "git"))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("parent of a 0600 entry created with mode %o, want 700", dirInfo.Mode().Perm())
	}

	h.write(rel, "local edit\n", 0o600)
	if out, _ := h.mustFail(1, "pull"); out != "differs    "+target+" (use -f to overwrite)\n" {
		t.Fatalf("pull differs: %q", out)
	}
	if out := h.mustRun("pull", "-n", "-f"); out != "would write "+target+"\n" {
		t.Fatalf("pull dry run: %q", out)
	}
	if got := h.read(rel); got != "local edit\n" {
		t.Fatal("dry run wrote the file")
	}
	if out := h.mustRun("pull", "-f"); out != "restored   "+target+"\n" {
		t.Fatalf("pull force: %q", out)
	}
	if got := h.read(rel); got != "[user]\n\tname = b\n" {
		t.Fatalf("forced content %q", got)
	}

	os.Chmod(live, 0o644)
	if out, _ := h.mustFail(1, "diff"); !strings.Contains(out, "M "+target+" (mode 0600 in store, 0644 live)") {
		t.Fatalf("diff mode: %q", out)
	}
	h.mustRun("push")
	if out := h.mustRun("diff"); out != "= "+target+"\n" {
		t.Fatalf("diff after mode push: %q", out)
	}

	if out := h.mustRun("rm", target); out != "removed "+target+"\n" {
		t.Fatalf("rm: %q", out)
	}
	if out := h.mustRun("diff"); out != "" {
		t.Fatalf("diff after rm: %q", out)
	}
	if _, errs := h.mustFail(1, "rm", target); !strings.Contains(errs, "not registered") {
		t.Fatalf("rm again: %q", errs)
	}
}

func TestAddSeveralFilesInOneRun(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	a := h.write(".bash_logout", "a\n", 0o644)
	b := h.write(".bashrc", "b\n", 0o644)
	c := h.write(".profile", "c\n", 0o600)
	gen := h.generation()
	out := h.mustRun("add", a, b, c)
	for _, want := range []string{"added ~/.bash_logout (mode 0644)", "added ~/.bashrc (mode 0644)", "added ~/.profile (mode 0600)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("multi add lacks %q: %q", want, out)
		}
	}
	if h.generation() != gen+1 {
		t.Fatalf("generation advanced by %d, want 1", h.generation()-gen)
	}
	d := h.write(".vimrc", "d\n", 0o644)
	e := h.write(".gitignore", "e\n", 0o644)
	gen = h.generation()
	out, errs := h.mustFail(1, "add", d, filepath.Join(h.home, "missing"), e)
	if !strings.Contains(out, "added ~/.vimrc") || !strings.Contains(out, "added ~/.gitignore") || !strings.Contains(errs, "missing") {
		t.Fatalf("partial add: out %q err %q", out, errs)
	}
	if h.generation() != gen+1 {
		t.Fatal("partial add must still save once")
	}
	if out := h.mustRun("ls"); strings.Count(out, "\n") != 6 {
		t.Fatalf("ls after adds: %q", out)
	}
}

func TestAddRefusesDuplicatesAndNonFiles(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	live := h.write(".bashrc", "x\n", 0o644)
	h.mustRun("add", live)
	_, errs := h.mustFail(1, "add", live)
	if !strings.Contains(errs, "already registered") || !strings.Contains(errs, "macfit push") {
		t.Fatalf("duplicate add: %q", errs)
	}
	if _, errs := h.mustFail(1, "add", filepath.Join(h.home, ".config")); !strings.Contains(errs, "not a regular file") {
		t.Fatalf("directory add: %q", errs)
	}
	link := filepath.Join(h.home, ".link")
	if err := os.Symlink(live, link); err != nil {
		t.Fatal(err)
	}
	if _, errs := h.mustFail(1, "add", link); !strings.Contains(errs, "not a regular file") {
		t.Fatalf("symlink add: %q", errs)
	}
	if _, errs := h.mustFail(1, "add", filepath.Join(h.home, "missing")); errs == "" {
		t.Fatal("missing file add must fail")
	}
	if out := h.mustRun("add", live, "-H", "b"); !strings.Contains(out, "added ~/.bashrc (mode 0644, host b)") {
		t.Fatalf("host-bound add: %q", out)
	}
	cfg := h.write(".config/git/config", "c\n", 0o644)
	if out := h.mustRun("add", cfg, "-l"); !strings.Contains(out, "added ~/.config/git/config") {
		t.Fatalf("literal add: %q", out)
	}
	if out := h.mustRun("push", cfg); out != "unchanged  ~/.config/git/config\n" {
		t.Fatalf("push by live path of a literal entry: %q", out)
	}
	if _, errs := h.mustFail(1, "push", "~/nothing"); !strings.Contains(errs, "not registered") {
		t.Fatalf("push unknown target: %q", errs)
	}
}

func TestPullRefusesSymlink(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	live := h.write(".bashrc", "real\n", 0o644)
	h.mustRun("add", live)
	other := h.write("elsewhere", "other\n", 0o644)
	if err := os.Remove(live); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, live); err != nil {
		t.Fatal(err)
	}
	out, _ := h.mustFail(1, "pull", "-f")
	if !strings.Contains(out, "symlink    ~/.bashrc") {
		t.Fatalf("pull onto symlink: %q", out)
	}
	info, err := os.Lstat(live)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced")
	}
	if got := h.read("elsewhere"); got != "other\n" {
		t.Fatal("symlink target was written through")
	}
}

func TestHostBoundEntriesWinOnTheirHost(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	live := h.write(".bashrc", "shared\n", 0o644)
	h.mustRun("add", live)
	h.write(".bashrc", "for a\n", 0o644)
	h.mustRun("add", live, "-H", "a")
	h.write(".bashrc", "for b\n", 0o644)
	h.mustRun("add", live, "-H", "b")

	os.Remove(live)
	h.mustRun("pull")
	if got := h.read(".bashrc"); got != "for a\n" {
		t.Fatalf("host a pulled %q", got)
	}
	if out := h.mustRun("diff"); out != "= ~/.bashrc\n" {
		t.Fatalf("host a diff: %q", out)
	}

	h.app.host = "c"
	os.Remove(live)
	h.mustRun("pull")
	if got := h.read(".bashrc"); got != "shared\n" {
		t.Fatalf("host c pulled %q", got)
	}
	h.app.host = "a"
	if out, _ := h.mustFail(1, "diff"); out != "M ~/.bashrc\n" {
		t.Fatalf("host a diff against shared content: %q", out)
	}

	out := h.mustRun("ls")
	if strings.Count(out, "~/.bashrc") != 3 {
		t.Fatalf("ls must list every variant: %q", out)
	}
	if out := h.mustRun("rm", live); out != "removed ~/.bashrc, host a\n" {
		t.Fatalf("rm picks this host's entry: %q", out)
	}
	if out := h.mustRun("rm", live, "-H", "b"); out != "removed ~/.bashrc, host b\n" {
		t.Fatalf("rm -H: %q", out)
	}
	if out := h.mustRun("rm", live); out != "removed ~/.bashrc\n" {
		t.Fatalf("rm falls back to the unbound entry: %q", out)
	}
}

func TestConflictCopyWarning(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	for _, n := range []string{"macfit 2.store", "macfit (1).store"} {
		if err := os.WriteFile(filepath.Join(filepath.Dir(h.store), n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, _, errs := h.run("ls")
	if !strings.Contains(errs, "sync conflict copies") || !strings.Contains(errs, "macfit 2.store") || !strings.Contains(errs, "macfit (1).store") {
		t.Fatalf("warning: %q", errs)
	}
}

func TestDiffRendersDriftInYellow(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	same := h.write(".profile", "same\n", 0o644)
	changed := h.write(".bashrc", "one\n", 0o644)
	gone := h.write(".vimrc", "gone\n", 0o644)
	h.mustRun("add", same, changed, gone)
	h.write(".bashrc", "two\n", 0o644)
	os.Remove(gone)

	defer color.SetEnabled(true)()
	code, out, _ := h.run("diff", "-V")
	if code != 1 {
		t.Fatalf("diff exit %d, want 1", code)
	}
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		plain := color.ClearCode(line)
		switch {
		case strings.HasPrefix(plain, "= "):
			if strings.Contains(line, "\x1b[") {
				t.Fatalf("= line is colored: %q", line)
			}
		default:
			if !strings.HasPrefix(line, yellow) {
				t.Fatalf("drift line is not yellow: %q", line)
			}
		}
	}
	for _, want := range []string{"M ~/.bashrc", "? ~/.vimrc", "-one", "+two", "@@ -1,1 +1,1 @@"} {
		if !strings.Contains(color.ClearCode(out), want) {
			t.Fatalf("diff -V lacks %q: %q", want, out)
		}
	}

	restore := color.SetEnabled(false)
	code, out, _ = h.run("diff", "-V")
	restore()
	if code != 1 || strings.Contains(out, "\x1b[") {
		t.Fatalf("color disabled: exit %d out %q", code, out)
	}
}

func TestUnifiedDiff(t *testing.T) {
	got := unifiedDiff("s", "l", []byte("a\nb\nc\n"), []byte("a\nx\nc\nd\n"))
	want := "--- s\n+++ l\n@@ -1,3 +1,4 @@\n a\n-b\n+x\n c\n+d\n"
	if got != want {
		t.Fatalf("diff\n got %q\nwant %q", got, want)
	}
	if got := unifiedDiff("s", "l", nil, []byte("new\n")); !strings.Contains(got, "+new\n") {
		t.Fatalf("empty store side: %q", got)
	}
}

func TestParseArgs(t *testing.T) {
	specs := []flagSpec{{"-H", "--host", true}, {"-l", "--literal", false}}
	flags, pos, err := parseArgs([]string{"a", "-H", "np10", "--literal", "b", "--", "-c"}, specs)
	if err != nil || flags["--host"] != "np10" || flags["--literal"] != "true" || strings.Join(pos, ",") != "a,b,-c" {
		t.Fatalf("parse: %v %v %v", flags, pos, err)
	}
	if _, _, err := parseArgs([]string{"--host"}, specs); err == nil {
		t.Fatal("missing value accepted")
	}
	if _, _, err := parseArgs([]string{"--literal=x"}, specs); err == nil {
		t.Fatal("value on a boolean flag accepted")
	}
	flags, _, _ = parseArgs([]string{"--host=x"}, specs)
	if flags["--host"] != "x" {
		t.Fatalf("inline value: %v", flags)
	}
}

func TestStoreEnvironmentAndFlagErrors(t *testing.T) {
	h := newHarness(t)
	h.app.storeEnv = filepath.Join(h.root, "synced", "env.store")
	if code, out, errs := h.runRaw("init", "-N"); code != 0 || !strings.Contains(out, "env.store") || strings.Contains(out, "remembered") {
		t.Fatalf("MACFIT_STORE: code %d out %q err %q", code, out, errs)
	}
	h.app.storeEnv = ""
	if code, _, errs := h.runRaw("ls"); code != 1 || !strings.Contains(errs, filepath.Join(".local", "share", "macfit", "macfit.store")) {
		t.Fatalf("default store path: code %d err %q", code, errs)
	}
	if code, _, _ := h.runRaw("ls", "--store"); code != 2 {
		t.Fatalf("--store without value: code %d", code)
	}
	if code, _, _ := h.run("init", "-N", "extra"); code != 2 {
		t.Fatalf("init with positional: code %d", code)
	}
}

func TestEmptyFileRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "-N")
	live := h.write(".hushlogin", "", 0o644)
	if out := h.mustRun("add", live); !strings.Contains(out, "added ~/.hushlogin") {
		t.Fatalf("add empty: %q", out)
	}
	if out := h.mustRun("diff"); out != "= ~/.hushlogin\n" {
		t.Fatalf("diff empty: %q", out)
	}
	if err := os.Remove(live); err != nil {
		t.Fatal(err)
	}
	if out := h.mustRun("pull"); out != "restored   ~/.hushlogin\n" {
		t.Fatalf("pull empty: %q", out)
	}
	if got := h.read(".hushlogin"); got != "" {
		t.Fatalf("restored empty file holds %q", got)
	}
	if out := h.mustRun("push"); out != "unchanged  ~/.hushlogin\n" {
		t.Fatalf("push empty: %q", out)
	}
}

// errReader makes readLine fail, proving a prompt was not reached.
func mustNotPrompt(t *testing.T) func(string) (string, error) {
	return func(string) (string, error) {
		t.Fatal("prompt reached unexpectedly")
		return "", errors.New("unreachable")
	}
}
