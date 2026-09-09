package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/queone/gkit/internal/lockbox"
)

// testKDF keeps passphrase derivation fast in tests.
var testKDF = lockbox.KDF{Time: 1, Memory: 8 * 1024, Threads: 1}

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
	icloud := filepath.Join(root, "icloud")
	if err := os.MkdirAll(icloud, 0o755); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, out: &bytes.Buffer{}, errb: &bytes.Buffer{}, root: root, home: home,
		store: filepath.Join(icloud, "macfit.store"), keys: &lockbox.MemoryKeyStore{}}
	h.app = &app{
		stdout: h.out, stderr: h.errb, keys: h.keys, env: testEnv(home), host: "a", kdf: testKDF, goos: "darwin",
		isTerminal: func() bool { return true },
		readSecret: func(string) ([]byte, error) { return []byte("pw"), nil },
	}
	return h
}

// run executes macfit against the harness store and returns exit code, stdout, stderr.
func (h *harness) run(args ...string) (int, string, string) {
	h.out.Reset()
	h.errb.Reset()
	code := h.app.run(append([]string{"-s", h.store}, args...))
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

func TestVersionAndHelp(t *testing.T) {
	h := newHarness(t)
	for _, arg := range []string{"--version", "-v", "v", "version"} {
		h.out.Reset()
		h.errb.Reset()
		if code := h.app.run([]string{arg}); code != 0 || h.out.String() != "macfit v1.0.0\n" || h.errb.Len() != 0 {
			t.Fatalf("%s: code %d stdout %q stderr %q", arg, code, h.out.String(), h.errb.String())
		}
	}
	h.out.Reset()
	if code := h.app.run(nil); code != 0 || !strings.Contains(h.out.String(), "\nUsage\n") {
		t.Fatalf("bare invocation: code %d out %q", code, h.out.String())
	}
	h.out.Reset()
	if code := h.app.run([]string{"help"}); code != 0 || !strings.Contains(h.out.String(), "macfit pull") {
		t.Fatalf("help: code %d out %q", code, h.out.String())
	}
	if code, _, errs := h.run("bogus"); code != 2 || !strings.Contains(errs, "unknown command") {
		t.Fatalf("unknown command: code %d stderr %q", code, errs)
	}
	if code, _, errs := h.run("add", "--nope", "x"); code != 2 || !strings.Contains(errs, "usage") {
		t.Fatalf("unknown flag: code %d stderr %q", code, errs)
	}
}

func TestPlatformGuard(t *testing.T) {
	h := newHarness(t)
	h.app.goos = "linux"
	for _, verb := range []string{"init", "add", "rm", "ls", "push", "pull", "diff"} {
		code, _, errs := h.run(verb)
		if code != 1 || !strings.Contains(errs, "macfit supports macOS only") {
			t.Fatalf("%s on linux: code %d stderr %q", verb, code, errs)
		}
	}
	h.out.Reset()
	if code := h.app.run([]string{"--version"}); code != 0 {
		t.Fatalf("--version on linux: code %d", code)
	}
	h.out.Reset()
	if code := h.app.run([]string{"help"}); code != 0 {
		t.Fatalf("help on linux: code %d", code)
	}
}

func TestInitCreatesStoreAndAnotherMacRecoversTheKey(t *testing.T) {
	h := newHarness(t)
	out := h.mustRun("init")
	if !strings.Contains(out, "store created at "+h.store) {
		t.Fatalf("init output: %q", out)
	}
	info, err := os.Stat(h.store)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store mode %o", info.Mode().Perm())
	}
	if len(h.keys.Keys) != 1 {
		t.Fatalf("keychain holds %d keys, want 1", len(h.keys.Keys))
	}
	var savedID string
	var savedKey []byte
	for id, k := range h.keys.Keys {
		savedID, savedKey = id, k
	}
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "already initialized") {
		t.Fatalf("second init: %q", errs)
	}

	other := newHarness(t)
	other.store = h.store
	out = other.mustRun("init")
	if !strings.Contains(out, "saved to the login keychain") {
		t.Fatalf("other Mac init: %q", out)
	}
	if got := other.keys.Keys[savedID]; !bytes.Equal(got, savedKey) {
		t.Fatal("other Mac recovered a different key")
	}
	other.mustRun("ls")

	wrong := newHarness(t)
	wrong.store = h.store
	wrong.app.readSecret = func(string) ([]byte, error) { return []byte("nope"), nil }
	if _, errs := wrong.mustFail(1, "init"); !strings.Contains(errs, "wrong passphrase") {
		t.Fatalf("wrong passphrase: %q", errs)
	}
	if len(wrong.keys.Keys) != 0 {
		t.Fatal("wrong passphrase must not save a key")
	}
	if _, errs := wrong.mustFail(1, "ls"); !strings.Contains(errs, "no key for") {
		t.Fatalf("ls without key: %q", errs)
	}
}

func TestInitRefusalsChangeNothing(t *testing.T) {
	h := newHarness(t)
	h.app.isTerminal = func() bool { return false }
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "init needs a terminal") {
		t.Fatalf("non-terminal: %q", errs)
	}
	if _, err := os.Stat(h.store); !os.IsNotExist(err) {
		t.Fatal("store created without a terminal")
	}
	h.app.isTerminal = func() bool { return true }

	missing := filepath.Join(h.root, "no-icloud", "macfit.store")
	h.store = missing
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "does not exist") {
		t.Fatalf("missing parent: %q", errs)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !os.IsNotExist(err) {
		t.Fatal("init created the missing parent directory")
	}
	h.store = filepath.Join(h.root, "icloud", "macfit.store")

	answers := [][]byte{[]byte("one"), []byte("two")}
	h.app.readSecret = func(string) ([]byte, error) {
		a := answers[0]
		answers = answers[1:]
		return a, nil
	}
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "do not match") {
		t.Fatalf("mismatch: %q", errs)
	}
	h.app.readSecret = func(string) ([]byte, error) { return []byte("  "), nil }
	if _, errs := h.mustFail(1, "init"); !strings.Contains(errs, "must not be empty") {
		t.Fatalf("empty: %q", errs)
	}
	if _, err := os.Stat(h.store); !os.IsNotExist(err) {
		t.Fatal("a refused init wrote a store")
	}
	if _, errs := h.mustFail(1, "ls"); !strings.Contains(errs, "run `macfit init`") {
		t.Fatalf("ls without store: %q", errs)
	}
}

func TestAddPushDiffPullRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init")
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

func TestAddRefusesDuplicatesAndNonFiles(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init")
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
	h.mustRun("init")
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
	h.mustRun("init")
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
	h.mustRun("init")
	copy := filepath.Join(filepath.Dir(h.store), "macfit 2.store")
	if err := os.WriteFile(copy, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, errs := h.run("ls")
	if !strings.Contains(errs, "conflict copies") || !strings.Contains(errs, "macfit 2.store") {
		t.Fatalf("warning: %q", errs)
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

func TestStoreFlagAndEnvironment(t *testing.T) {
	h := newHarness(t)
	h.app.storeEnv = filepath.Join(h.root, "icloud", "env.store")
	h.out.Reset()
	if code := h.app.run([]string{"init"}); code != 0 || !strings.Contains(h.out.String(), "env.store") {
		t.Fatalf("MACFIT_STORE: code %d out %q err %q", code, h.out.String(), h.errb.String())
	}
	h.app.storeEnv = ""
	h.errb.Reset()
	if code := h.app.run([]string{"ls"}); code != 1 || !strings.Contains(h.errb.String(), defaultStoreRel) {
		t.Fatalf("default store path: code %d err %q", code, h.errb.String())
	}
	if code := h.app.run([]string{"ls", "--store"}); code != 2 {
		t.Fatalf("--store without value: code %d", code)
	}
}

func TestEmptyFileRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init")
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

func TestHelpLayoutMatchesTheOtherUtilities(t *testing.T) {
	h := newHarness(t)
	for _, arg := range []string{"help", "-h", "-?", "--help", "h"} {
		h.out.Reset()
		if code := h.app.run([]string{arg}); code != 0 {
			t.Fatalf("%s: code %d", arg, code)
		}
		lines := strings.Split(h.out.String(), "\n")
		if lines[0] != "macfit v1.0.0" {
			t.Fatalf("%s: first line %q", arg, lines[0])
		}
		if lines[1] != "Keep Mac config files in one encrypted store and restore them on any Mac." {
			t.Fatalf("%s: second line %q", arg, lines[1])
		}
		last := -1
		for _, section := range []string{"\nOverview\n", "\nUsage\n", "\nOptions\n", "\nNotes\n"} {
			idx := strings.Index(h.out.String(), section)
			if idx < 0 || idx < last {
				t.Fatalf("%s: section %q missing or out of order", arg, strings.TrimSpace(section))
			}
			last = idx
		}
		if !strings.Contains(h.out.String(), "  -h, -?, --help     Show this help message and exit") {
			t.Fatalf("%s: help option line missing", arg)
		}
	}
}
