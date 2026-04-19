package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// --- test helpers ---

// setupHome creates a fresh tmp dir, points package-level home/hostname
// at it, forces osName=darwin, recomputes paths, and returns the tmp dir.
// Restores original state at t.Cleanup.
func setupHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	origHome := home
	origHost := hostname
	origOS := osName
	home = tmp
	hostname = "testhost"
	osName = "darwin"
	recomputePaths()
	t.Cleanup(func() {
		home = origHome
		hostname = origHost
		osName = origOS
		recomputePaths()
	})
	// Preseed the iCloud target so cloudBase -> iCloud symlink is creatable.
	if err := os.MkdirAll(filepath.Join(tmp, icloudTarget), 0755); err != nil {
		t.Fatalf("preseed iCloud: %v", err)
	}
	return tmp
}

// setupHealthyEnv builds on setupHome by creating the iCloud symlink,
// cloud memory + projects dirs, ~/.claude, and both local symlinks.
func setupHealthyEnv(t *testing.T) string {
	t.Helper()
	tmp := setupHome(t)
	if err := os.Symlink(icloudTarget, cloudBase); err != nil {
		t.Fatalf("cloudBase symlink: %v", err)
	}
	if err := os.MkdirAll(cloudMemory, 0755); err != nil {
		t.Fatalf("cloudMemory: %v", err)
	}
	if err := os.MkdirAll(cloudProjects, 0755); err != nil {
		t.Fatalf("cloudProjects: %v", err)
	}
	if err := os.MkdirAll(localClaude, 0755); err != nil {
		t.Fatalf("localClaude: %v", err)
	}
	if err := os.Symlink(cloudMemory, localMemory); err != nil {
		t.Fatalf("localMemory symlink: %v", err)
	}
	if err := os.Symlink(cloudProjects, localProjects); err != nil {
		t.Fatalf("localProjects symlink: %v", err)
	}
	return tmp
}

// withCwd chdir's to dir for the duration of the test.
func withCwd(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// runCmd invokes run() with captured stdout/stderr.
func runCmd(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	exit := run(args, &stdout, &stderr)
	return exit, stdout.String(), stderr.String()
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// --- AT1: status mode, healthy env, no perms file ---

func TestAT1_StatusAllOK(t *testing.T) {
	setupHealthyEnv(t)
	tmpCwd := t.TempDir() // cwd with no .claude/
	withCwd(t, tmpCwd)

	exit, stdout, _ := runCmd()
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s", exit, stdout)
	}
	if strings.Count(stdout, "OK:") != 5 {
		t.Errorf("want 5 OK lines; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "(no project permissions file in cwd)") {
		t.Errorf("missing 'no project permissions file' line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "env:") || !strings.Contains(stdout, "perms:") {
		t.Errorf("missing section headers:\n%s", stdout)
	}
}

// --- AT2: broken env, perms section still renders ---

func TestAT2_BrokenEnvPermsRenders(t *testing.T) {
	tmp := setupHome(t)
	// Deliberately do NOT create the iCloud symlink — env is broken.
	_ = tmp
	tmpCwd := t.TempDir()
	withCwd(t, tmpCwd)

	exit, stdout, _ := runCmd()
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	if !strings.Contains(stdout, "FAIL:") {
		t.Errorf("want FAIL line in env section:\n%s", stdout)
	}
	// Assert perms section header appears in stdout — proves env checks did
	// not short-circuit with os.Exit before perms rendering.
	if !strings.Contains(stdout, "perms:") {
		t.Errorf("perms section header missing — env short-circuited:\n%s", stdout)
	}
	if !strings.Contains(stdout, "(no project permissions file in cwd)") {
		t.Errorf("perms section body missing:\n%s", stdout)
	}
	// Ordering: env before perms in stdout.
	envIdx := strings.Index(stdout, "env:")
	permsIdx := strings.Index(stdout, "perms:")
	if envIdx < 0 || permsIdx < 0 || envIdx > permsIdx {
		t.Errorf("unexpected ordering: env@%d perms@%d", envIdx, permsIdx)
	}
}

// --- AT3: env -c and env --check produce consistent verify-only output ---

func TestAT3_EnvCheckShortAndLong(t *testing.T) {
	setupHealthyEnv(t)

	okRegex := regexp.MustCompile(`^CHECK: environment OK( \(.*\))?$`)

	for _, form := range []string{"-c", "--check"} {
		exit, stdout, _ := runCmd("env", form)
		if exit != 0 {
			t.Errorf("%s: exit = %d, want 0\n%s", form, exit, stdout)
		}
		// Locate the final CHECK line.
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		if len(lines) == 0 {
			t.Errorf("%s: no output", form)
			continue
		}
		last := lines[len(lines)-1]
		if !okRegex.MatchString(last) {
			t.Errorf("%s: final line %q does not match %s", form, last, okRegex)
		}
	}
}

func TestAT3_EnvCheckBroken(t *testing.T) {
	setupHome(t)
	for _, form := range []string{"-c", "--check"} {
		exit, _, _ := runCmd("env", form)
		if exit != 1 {
			t.Errorf("%s: exit = %d, want 1", form, exit)
		}
	}
}

// --- AT4: env -i and env --info print full design-philosophy text ---

func TestAT4_EnvInfo(t *testing.T) {
	uniquePhrases := []string{
		"PROBLEM",
		"Layer 1: Repo governance",
		"DESIGN PRINCIPLES",
		"HOW IT WORKS",
		"WHAT THIS DOES NOT DO",
		"RACE CONDITIONS",
	}
	for _, form := range []string{"-i", "--info"} {
		exit, stdout, _ := runCmd("env", form)
		if exit != 0 {
			t.Errorf("%s: exit %d", form, exit)
		}
		for _, p := range uniquePhrases {
			if !strings.Contains(stdout, p) {
				t.Errorf("%s: missing phrase %q", form, p)
			}
		}
	}
}

// --- AT5: env setup migrates memory and projects ---

func TestAT5_EnvSetupMigrates(t *testing.T) {
	tmp := setupHome(t)
	// iCloud symlink
	if err := os.Symlink(icloudTarget, cloudBase); err != nil {
		t.Fatal(err)
	}
	// Populated local dirs (no symlinks yet)
	mustWrite(t, filepath.Join(localMemory, "pref.md"), "foo")
	mustWrite(t, filepath.Join(localProjects, "repoA", "notes.md"), "bar")

	exit, stdout, _ := runCmd("env")
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stdout)
	}

	// Cloud dirs populated.
	if data, _ := os.ReadFile(filepath.Join(cloudMemory, "pref.md")); string(data) != "foo" {
		t.Errorf("memory not migrated: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(cloudProjects, "repoA", "notes.md")); string(data) != "bar" {
		t.Errorf("projects not migrated: %q", data)
	}
	// Local paths now symlinks.
	for _, local := range []string{localMemory, localProjects} {
		lfi, err := os.Lstat(local)
		if err != nil {
			t.Fatalf("lstat %s: %v", local, err)
		}
		if lfi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", local)
		}
	}
	_ = tmp
}

// --- AT5b: idempotence ---

func TestAT5b_EnvSetupIdempotent(t *testing.T) {
	setupHealthyEnv(t)

	// Snapshot cloud dir entries' modtime + size before.
	snapshot := func() map[string][2]any {
		m := map[string][2]any{}
		for _, root := range []string{cloudMemory, cloudProjects, localClaude} {
			filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				m[path] = [2]any{info.ModTime(), info.Size()}
				return nil
			})
		}
		return m
	}

	before := snapshot()

	exit, stdout, _ := runCmd("env")
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stdout)
	}

	// Output should contain only OK lines (no CREATED/MIGRATED/LINKED/REMOVED on a steady state).
	for _, bad := range []string{"MIGRATED:", "CREATED:", "LINKED:", "REMOVED:"} {
		if strings.Contains(stdout, bad) {
			t.Errorf("idempotent run produced %s:\n%s", bad, stdout)
		}
	}

	after := snapshot()
	for k, v := range before {
		a, ok := after[k]
		if !ok {
			t.Errorf("path disappeared: %s", k)
			continue
		}
		if a != v {
			t.Errorf("path %s changed: before %v, after %v", k, v, a)
		}
	}
}

// --- AT6: perms init fresh dir ---

func TestAT6_PermsInitFresh(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)

	exit, _, stderr := runCmd("perms", "init")
	if exit != 0 {
		t.Fatalf("exit %d; stderr=%s", exit, stderr)
	}
	data, err := os.ReadFile(filepath.Join(cwd, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	perms, _ := got["permissions"].(map[string]any)
	allow, _ := perms["allow"].([]any)
	if len(allow) != 2 {
		t.Errorf("allow len = %d, want 2: %v", len(allow), allow)
	}
	// Expect only the `permissions` key at top level.
	if len(got) != 1 {
		t.Errorf("top-level keys = %d, want 1: %v", len(got), got)
	}
	// Expect only the `allow` key under permissions.
	if len(perms) != 1 {
		t.Errorf("permissions keys = %d, want 1: %v", len(perms), perms)
	}
}

// --- AT7: perms init merges preserving other keys + dedup ---

func TestAT7_PermsInitMerge(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)

	initial := `{"permissions":{"allow":["Bash(go list *)","Bash(foo)"]},"spinnerTipsEnabled":false}`
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"), initial)

	exit, _, stderr := runCmd("perms", "init")
	if exit != 0 {
		t.Fatalf("exit %d; stderr=%s", exit, stderr)
	}
	data, err := os.ReadFile(filepath.Join(cwd, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	perms := merged["permissions"].(map[string]any)
	allow := perms["allow"].([]any)
	got := make([]string, len(allow))
	for i, a := range allow {
		got[i] = a.(string)
	}
	// Expect dedup: Bash(go list *) once, Bash(foo) retained, Bash(staticcheck *) appended.
	counts := map[string]int{}
	for _, e := range got {
		counts[e]++
	}
	if counts["Bash(go list *)"] != 1 {
		t.Errorf("Bash(go list *) count = %d, want 1: %v", counts["Bash(go list *)"], got)
	}
	if counts["Bash(foo)"] != 1 {
		t.Errorf("Bash(foo) count = %d, want 1: %v", counts["Bash(foo)"], got)
	}
	if counts["Bash(staticcheck *)"] != 1 {
		t.Errorf("Bash(staticcheck *) count = %d, want 1: %v", counts["Bash(staticcheck *)"], got)
	}
	// Verify spinnerTipsEnabled preserved with value false.
	spin, ok := merged["spinnerTipsEnabled"]
	if !ok {
		t.Fatal("spinnerTipsEnabled key missing after merge")
	}
	if spin != false {
		t.Errorf("spinnerTipsEnabled = %v, want false", spin)
	}
	// Verify two-space indent (formatting normalized).
	if !strings.Contains(string(data), "\n  ") {
		t.Errorf("output not indented with two spaces:\n%s", data)
	}
}

// --- AT8: perms init unknown profile ---

func TestAT8_PermsInitUnknownProfile(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)

	for _, form := range [][]string{{"perms", "init", "-p", "doesnotexist"}, {"perms", "init", "--profile", "doesnotexist"}} {
		exit, stdout, stderr := runCmd(form...)
		if exit != 1 {
			t.Errorf("%v: exit %d, want 1; stdout=%q", form, exit, stdout)
		}
		if !strings.Contains(stderr, "doesnotexist") {
			t.Errorf("%v: stderr missing profile name: %q", form, stderr)
		}
	}
	// Assert no file created.
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "settings.json")); err == nil {
		t.Error("settings.json was created despite unknown profile")
	}
}

// --- AT9: perms init dry-run ---

func TestAT9_PermsInitDryRun(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)

	for _, form := range [][]string{{"perms", "init", "-n"}, {"perms", "init", "--dry-run"}} {
		exit, stdout, _ := runCmd(form...)
		if exit != 0 {
			t.Errorf("%v: exit %d", form, exit)
		}
		if !strings.Contains(stdout, "Bash(go list *)") {
			t.Errorf("%v: stdout missing intended JSON: %q", form, stdout)
		}
		if _, err := os.Stat(filepath.Join(cwd, ".claude", "settings.json")); err == nil {
			t.Errorf("%v: file created despite dry-run", form)
		}
	}
}

// --- AT10: perms list alphabetical + override dir additive ---

func TestAT10_PermsListAlphabetical(t *testing.T) {
	setupHome(t)

	// No override dir yet.
	exit, stdout, _ := runCmd("perms", "list")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if strings.TrimSpace(stdout) != "go" {
		t.Errorf("list without override = %q, want %q", stdout, "go\n")
	}

	// Add override.
	mustWrite(t, filepath.Join(permsStarters, "custom.json"), `{"permissions":{"allow":["Bash(echo)"]}}`)
	exit, stdout, _ = runCmd("perms", "list")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	got := strings.Split(strings.TrimSpace(stdout), "\n")
	want := []string{"custom", "go"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("list with override = %v, want %v", got, want)
	}
}

// --- AT11: override shadowing by filename stem ---

func TestAT11_OverrideShadowing(t *testing.T) {
	setupHome(t)
	overrideContent := `{"permissions":{"allow":["Bash(OVERRIDE *)"]}}`
	mustWrite(t, filepath.Join(permsStarters, "go.json"), overrideContent)

	exit, stdout, _ := runCmd("perms", "show", "go")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if !strings.Contains(stdout, "Bash(OVERRIDE *)") {
		t.Errorf("override not shadowing: %q", stdout)
	}
	if strings.Contains(stdout, "Bash(staticcheck *)") {
		t.Errorf("embedded profile leaked through override: %q", stdout)
	}
}

// --- AT12: malformed override file skipped with stderr warning ---

func TestAT12_MalformedOverride(t *testing.T) {
	setupHome(t)
	mustWrite(t, filepath.Join(permsStarters, "broken.json"), "{not json")

	exit, stdout, stderr := runCmd("perms", "list")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	if !strings.Contains(stderr, "WARNING:") {
		t.Errorf("stderr missing WARNING: %q", stderr)
	}
	if !strings.Contains(stderr, "broken") {
		t.Errorf("stderr doesn't name broken profile: %q", stderr)
	}
	names := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, n := range names {
		if n == "broken" {
			t.Errorf("broken listed in profiles: %q", stdout)
		}
	}
	foundGo := false
	for _, n := range names {
		if n == "go" {
			foundGo = true
		}
	}
	if !foundGo {
		t.Errorf("go not listed despite malformed override: %q", stdout)
	}

	// perms show broken → exit 1.
	exit, _, _ = runCmd("perms", "show", "broken")
	if exit != 1 {
		t.Errorf("show broken exit %d, want 1", exit)
	}
}

// --- AT13: perms show go round-trips and contains Bash(go list *) ---

func TestAT13_PermsShowGo(t *testing.T) {
	setupHome(t)
	exit, stdout, _ := runCmd("perms", "show", "go")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(stdout), &v); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, stdout)
	}
	// Re-marshal round trip.
	if _, err := json.Marshal(v); err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	if !strings.Contains(stdout, "Bash(go list *)") {
		t.Errorf("missing expected entry: %s", stdout)
	}
}

// --- AT14: WARN on Bash(python3:*) ---

func TestAT14_WarnPython3(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.local.json"),
		`{"permissions":{"allow":["Bash(python3:*)"]}}`)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("default exit %d, want 0", exit)
	}
	if !strings.Contains(stdout, "settings.local.json:") {
		t.Errorf("missing source-file prefix: %q", stdout)
	}
	if !strings.Contains(stdout, "WARN") || !strings.Contains(stdout, "python3") {
		t.Errorf("missing WARN/python3: %q", stdout)
	}

	for _, form := range [][]string{{"perms", "check", "-s"}, {"perms", "check", "--strict"}} {
		exit, _, _ := runCmd(form...)
		if exit != 1 {
			t.Errorf("%v: exit %d, want 1", form, exit)
		}
	}
}

// --- AT15: INFO on Bash(ls:*) auto-allow ---

func TestAT15_InfoLsAutoAllow(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"),
		`{"permissions":{"allow":["Bash(ls:*)"]}}`)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("exit %d, want 0", exit)
	}
	if !strings.Contains(stdout, "settings.json:") {
		t.Errorf("missing source-file prefix: %q", stdout)
	}
	if !strings.Contains(stdout, "INFO") || !strings.Contains(stdout, "ls") {
		t.Errorf("missing INFO/ls: %q", stdout)
	}
	if strings.Contains(stdout, "WARN") {
		t.Errorf("unexpected WARN: %q", stdout)
	}

	// strict doesn't change exit for pure INFO.
	exit, _, _ = runCmd("perms", "check", "-s")
	if exit != 0 {
		t.Errorf("strict exit with only INFO = %d, want 0", exit)
	}
}

// --- AT16: INFO for absolute path in body ---

func TestAT16_InfoAbsolutePath(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"),
		`{"permissions":{"allow":["Bash(find /Users/tek1/code/oldrepo -name foo)"]}}`)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("exit %d, want 0", exit)
	}
	if !strings.Contains(stdout, "settings.json:") {
		t.Errorf("missing source-file prefix: %q", stdout)
	}
	// Must emit the absolute-path INFO specifically.
	if !strings.Contains(stdout, "absolute path") {
		t.Errorf("missing absolute-path INFO: %q", stdout)
	}
	if strings.Contains(stdout, "WARN") {
		t.Errorf("unexpected WARN: %q", stdout)
	}
}

// --- AT16b: same offending entry in both files → both lines ---

func TestAT16b_SourceFileAttributionBothFiles(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	content := `{"permissions":{"allow":["Bash(python3:*)"]}}`
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"), content)
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.local.json"), content)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("exit %d, want 0", exit)
	}
	if !strings.Contains(stdout, "settings.json:") {
		t.Errorf("missing settings.json prefix: %q", stdout)
	}
	if !strings.Contains(stdout, "settings.local.json:") {
		t.Errorf("missing settings.local.json prefix: %q", stdout)
	}
}

// --- AT16c: longest-prefix matching across go run, go list, git status ---

func TestAT16c_LongestPrefixMatching(t *testing.T) {
	cases := []struct {
		entry     string
		wantWarn  bool
		wantInfo  bool
		wantSilnt bool
	}{
		{"Bash(go run *)", true, false, false},
		{"Bash(go list *)", false, false, true},  // no head match, no abs path
		{"Bash(git status)", false, true, false}, // head=git status → INFO
	}
	for _, tc := range cases {
		t.Run(tc.entry, func(t *testing.T) {
			setupHome(t)
			cwd := t.TempDir()
			withCwd(t, cwd)
			mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"),
				`{"permissions":{"allow":["`+tc.entry+`"]}}`)

			exit, stdout, _ := runCmd("perms", "check")
			_ = exit
			hasWarn := strings.Contains(stdout, "WARN")
			hasInfo := strings.Contains(stdout, "INFO")
			if hasWarn != tc.wantWarn {
				t.Errorf("WARN: got %v want %v; stdout=%q", hasWarn, tc.wantWarn, stdout)
			}
			if hasInfo != tc.wantInfo {
				t.Errorf("INFO: got %v want %v; stdout=%q", hasInfo, tc.wantInfo, stdout)
			}
			if tc.wantSilnt && stdout != "" {
				t.Errorf("expected silent, got: %q", stdout)
			}
		})
	}
}

// --- AT16d: malformed-shape entries pass silently ---

func TestAT16d_MalformedShapeSilent(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"),
		`{"permissions":{"allow":["Bash(unclosed","Bash()","not-a-wrapper"]}}`)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("exit %d, want 0", exit)
	}
	if stdout != "" {
		t.Errorf("expected silent, got: %q", stdout)
	}
}

// --- AT17: non-Bash patterns pass silently ---

func TestAT17_NonBashPatternsSilent(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	content := `{"permissions":{"allow":["Read(**)","Write(**)","WebFetch(domain:example.com)","mcp__github__*"]}}`
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"), content)

	for _, flag := range []string{"", "-s", "--strict"} {
		args := []string{"perms", "check"}
		if flag != "" {
			args = append(args, flag)
		}
		exit, stdout, _ := runCmd(args...)
		if exit != 0 {
			t.Errorf("%s: exit %d, want 0", flag, exit)
		}
		if stdout != "" {
			t.Errorf("%s: expected silent, got: %q", flag, stdout)
		}
	}
}

// --- AT18: clean Bash-only allowlist produces nothing ---

func TestAT18_CleanAllowlistSilent(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	content := `{"permissions":{"allow":["Bash(go list *)","Bash(staticcheck *)","Bash(go build)"]}}`
	mustWrite(t, filepath.Join(cwd, ".claude", "settings.json"), content)

	exit, stdout, _ := runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("exit %d, want 0", exit)
	}
	if stdout != "" {
		t.Errorf("expected silent, got: %q", stdout)
	}
}

// --- AT19: settings.local.json byte-identical after perms init ---

func TestAT19_LocalJsonUntouched(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)
	localPath := filepath.Join(cwd, ".claude", "settings.local.json")
	mustWrite(t, localPath, `{"permissions":{"allow":["Bash(something)"]}}`)

	before := fileSHA256(t, localPath)

	exit, _, _ := runCmd("perms", "init")
	if exit != 0 {
		t.Fatalf("exit %d", exit)
	}

	after := fileSHA256(t, localPath)
	if before != after {
		t.Errorf("settings.local.json modified: sha256 before=%s after=%s", before, after)
	}
}

// --- AT20: conflict preservation on migration ---

func TestAT20_ConflictPreservation(t *testing.T) {
	setupHome(t)
	if err := os.Symlink(icloudTarget, cloudBase); err != nil {
		t.Fatal(err)
	}
	// Diverging content at local vs cloud.
	mustWrite(t, filepath.Join(localMemory, "shared.md"), "local-version")
	mustWrite(t, filepath.Join(cloudMemory, "shared.md"), "cloud-version")

	exit, stdout, _ := runCmd("env")
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stdout)
	}
	// Conflict file preserved.
	got, err := os.ReadFile(filepath.Join(cloudMemory, "shared.md.conflict-testhost"))
	if err != nil {
		t.Fatalf("conflict file missing: %v", err)
	}
	if string(got) != "local-version" {
		t.Errorf("conflict content = %q, want local-version", got)
	}
	// Cloud file untouched.
	cloudGot, _ := os.ReadFile(filepath.Join(cloudMemory, "shared.md"))
	if string(cloudGot) != "cloud-version" {
		t.Errorf("cloud overwritten: %q", cloudGot)
	}
	if !strings.Contains(stdout, "CONFLICT:") {
		t.Errorf("missing CONFLICT: line: %s", stdout)
	}
}

// --- AT21: env conflicts subcommand ---

func TestAT21_EnvConflicts(t *testing.T) {
	setupHealthyEnv(t)

	// Clean state.
	exit, stdout, _ := runCmd("env", "conflicts")
	if exit != 0 {
		t.Errorf("clean: exit %d", exit)
	}
	if !strings.Contains(stdout, "No conflict files") {
		t.Errorf("clean: %q", stdout)
	}

	// Seed one conflict file.
	mustWrite(t, filepath.Join(cloudMemory, "a.md.conflict-testhost"), "x")
	exit, stdout, _ = runCmd("env", "conflicts")
	if exit != 0 {
		t.Errorf("with conflict: exit %d", exit)
	}
	if !strings.Contains(stdout, "a.md.conflict-testhost") {
		t.Errorf("missing conflict path: %q", stdout)
	}
	if !strings.Contains(stdout, "resolve") {
		t.Errorf("missing resolution hint: %q", stdout)
	}
}

// --- AT22: platform gate on non-darwin ---

func TestAT22_NonDarwinGate(t *testing.T) {
	setupHome(t)
	osName = "linux"

	// env subcommands fail with consistent message.
	for _, args := range [][]string{{"env"}, {"env", "-c"}, {"env", "--check"}} {
		exit, _, stderr := runCmd(args...)
		if exit != 1 {
			t.Errorf("%v: exit %d, want 1", args, exit)
		}
		if !strings.Contains(stderr, "requires macOS") {
			t.Errorf("%v: stderr missing platform message: %q", args, stderr)
		}
	}

	// perms subcommands run normally (list, show, check in cwd).
	cwd := t.TempDir()
	withCwd(t, cwd)
	exit, _, _ := runCmd("perms", "list")
	if exit != 0 {
		t.Errorf("perms list on non-darwin: exit %d, want 0", exit)
	}
	exit, _, _ = runCmd("perms", "show", "go")
	if exit != 0 {
		t.Errorf("perms show on non-darwin: exit %d, want 0", exit)
	}
	exit, _, _ = runCmd("perms", "check")
	if exit != 0 {
		t.Errorf("perms check on non-darwin: exit %d, want 0", exit)
	}

	// Default status on non-darwin: env section skipped, perms section renders.
	exit, stdout, _ := runCmd()
	if exit != 0 {
		t.Errorf("status on non-darwin with no perms: exit %d, want 0", exit)
	}
	if !strings.Contains(stdout, "(skipped: non-darwin host)") {
		t.Errorf("missing skipped marker: %q", stdout)
	}
	if !strings.Contains(stdout, "perms:") {
		t.Errorf("missing perms section: %q", stdout)
	}
}

// --- AT23: version/help + short-first convention ---

func TestAT23_VersionAndHelp(t *testing.T) {
	setupHome(t)

	for _, form := range []string{"-v", "--version"} {
		exit, stdout, _ := runCmd(form)
		if exit != 0 {
			t.Errorf("%s: exit %d", form, exit)
		}
		if !strings.Contains(stdout, "claudecfg 1.0.0") {
			t.Errorf("%s: version line wrong: %q", form, stdout)
		}
	}

	for _, form := range []string{"-h", "--help"} {
		exit, stdout, _ := runCmd(form)
		if exit != 0 {
			t.Errorf("%s: exit %d", form, exit)
		}
		// Both subcommand groups mentioned.
		if !strings.Contains(stdout, "env") || !strings.Contains(stdout, "perms") {
			t.Errorf("%s: help missing subcommand groups: %q", form, stdout)
		}
		// Short-form before long-form for every flag the top-level help lists.
		// Flags shown: -v (--version), -h (--help).
		if !strings.Contains(stdout, "-v (--version)") || !strings.Contains(stdout, "-h (--help)") {
			t.Errorf("%s: help missing short-first convention: %q", form, stdout)
		}
	}

	// env help mentions "setup (mutating" and points to top-level claudecfg.
	exit, stdout, _ := runCmd("env", "-h")
	if exit != 0 {
		t.Errorf("env -h exit %d", exit)
	}
	if !strings.Contains(stdout, "setup (mutating") {
		t.Errorf("env help missing mutation callout: %q", stdout)
	}
	if !strings.Contains(stdout, "claudecfg") {
		t.Errorf("env help missing reference to top-level claudecfg: %q", stdout)
	}
}

// --- AT25: every flag has both short and long forms ---

func TestAT25_FlagFormsEquivalent(t *testing.T) {
	setupHome(t)

	// Find flags in help output of each subcommand, matching the pattern:
	//   -X (--long)    description
	flagRe := regexp.MustCompile(`-(\w) \((--\w[\w-]*)\)`)

	_, topHelp, _ := runCmd("-h")
	_, envHelp, _ := runCmd("env", "-h")
	_, permsHelp, _ := runCmd("perms", "-h")

	for _, block := range []string{topHelp, envHelp, permsHelp} {
		matches := flagRe.FindAllStringSubmatch(block, -1)
		if len(matches) == 0 {
			t.Errorf("no short/long flag pairs found in help:\n%s", block)
		}
		for _, m := range matches {
			short, long := "-"+m[1], m[2]
			if short == "" || long == "" {
				t.Errorf("incomplete pair: %v", m)
			}
		}
	}

	// Functional equivalence for a representative pair: -c vs --check.
	setupHealthyEnv(t)
	exitShort, _, _ := runCmd("env", "-c")
	exitLong, _, _ := runCmd("env", "--check")
	if exitShort != exitLong {
		t.Errorf("env -c exit %d != env --check exit %d", exitShort, exitLong)
	}
}

// --- AT25b: unknown subcommand/flag routing ---

func TestAT25b_UnknownCommands(t *testing.T) {
	setupHome(t)
	cwd := t.TempDir()
	withCwd(t, cwd)

	// Top-level unknown → top-level help, exit 1.
	exit, stdout, stderr := runCmd("foo")
	if exit != 1 {
		t.Errorf("'foo': exit %d, want 1", exit)
	}
	if stdout != "" {
		t.Errorf("'foo': stdout not empty: %q", stdout)
	}
	if !strings.Contains(stderr, "Unknown command: foo") {
		t.Errorf("'foo': stderr missing error: %q", stderr)
	}
	if !strings.Contains(stderr, "env") || !strings.Contains(stderr, "perms") {
		t.Errorf("'foo': stderr missing top-level help: %q", stderr)
	}

	// Nested unknown under perms → perms help.
	exit, stdout, stderr = runCmd("perms", "foo")
	if exit != 1 {
		t.Errorf("'perms foo': exit %d, want 1", exit)
	}
	if stdout != "" {
		t.Errorf("'perms foo': stdout not empty: %q", stdout)
	}
	if !strings.Contains(stderr, "Unknown command: foo") {
		t.Errorf("'perms foo': stderr missing error: %q", stderr)
	}
	// perms help must mention init/list/show/check (perms-specific subcommands).
	if !strings.Contains(stderr, "init") || !strings.Contains(stderr, "check") {
		t.Errorf("'perms foo': stderr missing perms help: %q", stderr)
	}

	// env --bogus → env help + unknown flag error.
	exit, stdout, stderr = runCmd("env", "--bogus")
	if exit != 1 {
		t.Errorf("'env --bogus': exit %d, want 1", exit)
	}
	if stdout != "" {
		t.Errorf("'env --bogus': stdout not empty: %q", stdout)
	}
	if !strings.Contains(stderr, "Unknown flag: --bogus") {
		t.Errorf("'env --bogus': stderr missing error: %q", stderr)
	}
	if !strings.Contains(stderr, "conflicts") {
		t.Errorf("'env --bogus': stderr missing env help: %q", stderr)
	}

	// No filesystem mutations.
	if _, err := os.Stat(filepath.Join(cwd, ".claude")); err == nil {
		t.Error(".claude created after unknown command")
	}
}

// --- AT24: repo state post-implementation ---

func TestAT24_RepoStatePostImpl(t *testing.T) {
	repo := repoRoot(t)
	// (a) cmd/claude-env/ does not exist
	if _, err := os.Stat(filepath.Join(repo, "cmd", "claude-env")); !os.IsNotExist(err) {
		t.Errorf("cmd/claude-env still exists: err=%v", err)
	}
	// (c) grep for live 'claude-env' refs outside CHANGELOG.md, docs/, .git/,
	// and .claude/ (user-local machine config, per AC24 spirit).
	thisTestFile := filepath.Join(repo, "cmd", "claudecfg", "main_test.go")
	_ = filepath.Walk(repo, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "docs" || base == ".claude" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "CHANGELOG.md" {
			return nil
		}
		if path == thisTestFile {
			return nil
		}
		if info.Size() > 1<<20 {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if bytes.Contains(data, []byte("claude-env")) {
			t.Errorf("stale 'claude-env' reference in %s", path)
		}
		return nil
	})
}

// --- AT26: AGENTS.md contains the lifted flag-convention rule ---

func TestAT26_AgentsMdFlagConventionRule(t *testing.T) {
	repo := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	wantPhrase := "one-letter short form"
	if !bytes.Contains(data, []byte(wantPhrase)) {
		t.Errorf("AGENTS.md missing flag-convention rule (%q)", wantPhrase)
	}
	// Must be under Project Rules section.
	content := string(data)
	projIdx := strings.Index(content, "## Project Rules")
	phraseIdx := strings.Index(content, wantPhrase)
	if projIdx < 0 || phraseIdx < 0 || phraseIdx < projIdx {
		t.Errorf("flag-convention rule not in Project Rules section")
	}
}

// --- helpers for repo-state tests ---

func repoRoot(t *testing.T) string {
	t.Helper()
	// cmd/claudecfg/main_test.go → ../../
	here, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(here, "..", ".."))
}
