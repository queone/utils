// claudecfg manages Claude Code configuration: iCloud-backed memory portability
// (env subcommands) and project .claude/settings.json permissions seeding and
// auditing (perms subcommands). With no arguments it runs a read-only status.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/queone/utils/internal/color"
)

const (
	programName    = "claudecfg"
	programVersion = "1.0.0"
)

//go:embed profiles/*.json
var embeddedProfiles embed.FS

// osName is the OS identifier used by the platform gate. A variable (not a
// constant) so tests can override it without cross-compilation.
var osName = runtime.GOOS

var (
	home          string
	hostname      string
	icloudTarget  string
	cloudBase     string
	cloudClaude   string
	cloudMemory   string
	cloudProjects string
	localClaude   string
	localMemory   string
	localProjects string
	permsStarters string
)

func init() {
	home = os.Getenv("HOME")
	hostname = getHostname()
	icloudTarget = "Library/Mobile Documents/com~apple~CloudDocs"
	recomputePaths()
}

// recomputePaths re-derives all path variables from home + hostname. Tests
// override home and hostname then call this to refresh the derived paths.
func recomputePaths() {
	cloudBase = filepath.Join(home, "data")
	cloudClaude = filepath.Join(cloudBase, "etc", "claude")
	cloudMemory = filepath.Join(cloudClaude, "memory")
	cloudProjects = filepath.Join(cloudClaude, hostname, "projects")
	localClaude = filepath.Join(home, ".claude")
	localMemory = filepath.Join(localClaude, "memory")
	localProjects = filepath.Join(localClaude, "projects")
	permsStarters = filepath.Join(cloudClaude, "perms-starters")
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runStatus(stdout)
	}
	switch args[0] {
	case "-v", "--version":
		fmt.Fprintf(stdout, "%s %s\n", programName, programVersion)
		return 0
	case "-h", "--help":
		fmt.Fprint(stdout, topLevelHelp())
		return 0
	case "env":
		return runEnv(args[1:], stdout, stderr)
	case "perms":
		return runPerms(args[1:], stdout, stderr)
	default:
		if strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "Unknown flag: %s\n", args[0])
		} else {
			fmt.Fprintf(stderr, "Unknown command: %s\n", args[0])
		}
		fmt.Fprint(stderr, topLevelHelp())
		return 1
	}
}

// --- status mode ---

func runStatus(stdout io.Writer) int {
	fmt.Fprintln(stdout, "env:")
	envExit := 0
	if osName != "darwin" {
		fmt.Fprintln(stdout, "  (skipped: non-darwin host)")
	} else {
		results := collectEnvChecks()
		for _, r := range results {
			if r.ok {
				fmt.Fprintf(stdout, "  OK: %s%s\n", r.name, optDetail(r.detail))
			} else {
				fmt.Fprintf(stdout, "  FAIL: %s — %s\n", r.name, r.detail)
				envExit = 1
			}
		}
		if cFiles, _ := findConflicts(cloudMemory, cloudProjects); len(cFiles) > 0 {
			fmt.Fprintf(stdout, "  NOTE: %d unresolved conflict file(s) — run 'claudecfg env conflicts' to list\n", len(cFiles))
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "perms:")
	permsRendered := renderPermsStatus(stdout)
	_ = permsRendered
	return envExit
}

func optDetail(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

// renderPermsStatus runs a non-strict perms check against the cwd's
// .claude/settings*.json files, writing output to w. Returns true if any
// settings file was found.
func renderPermsStatus(w io.Writer) bool {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(w, "  (cannot read cwd: %v)\n", err)
		return false
	}
	files := []string{
		filepath.Join(cwd, ".claude", "settings.json"),
		filepath.Join(cwd, ".claude", "settings.local.json"),
	}
	any := false
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			any = true
		}
	}
	if !any {
		fmt.Fprintln(w, "  (no project permissions file in cwd)")
		return false
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		checkPermsFile(w, filepath.Base(f), data, false)
	}
	return true
}

// --- env subcommand ---

type envCheckResult struct {
	name   string
	ok     bool
	detail string
}

func collectEnvChecks() []envCheckResult {
	var out []envCheckResult

	// 1. iCloud base symlink
	if ok, detail := checkCloudBase(); ok {
		out = append(out, envCheckResult{name: "iCloud base", ok: true, detail: detail})
	} else {
		out = append(out, envCheckResult{name: "iCloud base", ok: false, detail: detail})
	}

	// 2. cloud memory dir
	out = append(out, checkCloudDir(cloudMemory, "cloud memory dir"))
	// 3. cloud projects dir
	out = append(out, checkCloudDir(cloudProjects, "cloud projects dir"))
	// 4. local memory symlink
	out = append(out, checkLocalSymlink(localMemory, cloudMemory, "local memory symlink"))
	// 5. local projects symlink
	out = append(out, checkLocalSymlink(localProjects, cloudProjects, "local projects symlink"))

	return out
}

func checkCloudBase() (bool, string) {
	target, err := os.Readlink(cloudBase)
	if err != nil {
		return false, fmt.Sprintf("%s is not a symlink", cloudBase)
	}
	resolved := resolvePath(cloudBase, target)
	expected := filepath.Clean(filepath.Join(home, icloudTarget))
	if resolved != expected {
		return false, fmt.Sprintf("%s -> %s (expected %s)", cloudBase, resolved, expected)
	}
	return true, fmt.Sprintf("%s -> %s", cloudBase, resolved)
}

func checkCloudDir(path, name string) envCheckResult {
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		return envCheckResult{name: name, ok: false, detail: fmt.Sprintf("%s does not exist", path)}
	}
	return envCheckResult{name: name, ok: true, detail: path}
}

func checkLocalSymlink(localPath, expectedTarget, name string) envCheckResult {
	lfi, err := os.Lstat(localPath)
	if err != nil {
		return envCheckResult{name: name, ok: false, detail: fmt.Sprintf("%s does not exist", localPath)}
	}
	if lfi.Mode()&os.ModeSymlink == 0 {
		return envCheckResult{name: name, ok: false, detail: fmt.Sprintf("%s is not a symlink", localPath)}
	}
	target, err := os.Readlink(localPath)
	if err != nil {
		return envCheckResult{name: name, ok: false, detail: fmt.Sprintf("cannot read %s: %v", localPath, err)}
	}
	if resolvePath(localPath, target) != filepath.Clean(expectedTarget) {
		return envCheckResult{name: name, ok: false, detail: fmt.Sprintf("%s -> %s (expected %s)", localPath, target, expectedTarget)}
	}
	return envCheckResult{name: name, ok: true, detail: fmt.Sprintf("%s -> %s", localPath, expectedTarget)}
}

func runEnv(args []string, stdout, stderr io.Writer) int {
	if osName != "darwin" {
		fmt.Fprintln(stderr, "FAIL: claudecfg env requires macOS (uses iCloud for sync)")
		return 1
	}
	if len(args) == 0 {
		return runEnvSetup(stdout, false)
	}
	switch args[0] {
	case "-c", "--check":
		return runEnvSetup(stdout, true)
	case "-i", "--info":
		fmt.Fprint(stdout, envInfoText())
		return 0
	case "-h", "--help":
		fmt.Fprint(stdout, envHelp())
		return 0
	case "conflicts":
		return runEnvConflicts(stdout)
	default:
		if strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "Unknown flag: %s\n", args[0])
		} else {
			fmt.Fprintf(stderr, "Unknown command: %s\n", args[0])
		}
		fmt.Fprint(stderr, envHelp())
		return 1
	}
}

func runEnvSetup(stdout io.Writer, checkOnly bool) int {
	fmt.Fprintln(stdout, programName+": verifying Claude Code environment")
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, color.Blu("iCloud base:"))
	if !verifyCloudBase(stdout) {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Cannot proceed: ~/data must be a symlink to iCloud CloudDrive.")
		fmt.Fprintf(stdout, "Expected: cd %s && ln -s '%s' data\n", home, icloudTarget)
		return 1
	}

	if hostname == "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "FAIL: cannot determine hostname")
		return 1
	}

	if fi, err := os.Stat(localClaude); err != nil || !fi.IsDir() {
		if checkOnly {
			fmt.Fprintf(stdout, "\nFAIL: %s does not exist\n", localClaude)
			return 1
		}
		if err := os.MkdirAll(localClaude, 0755); err != nil {
			fmt.Fprintf(stdout, "\nFAIL: cannot create %s: %v\n", localClaude, err)
			return 1
		}
		fmt.Fprintf(stdout, "\n  CREATED: %s\n", localClaude)
	}

	errors := 0

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, color.Blu("Global memory:"))
	if !setupLink(stdout, localMemory, cloudMemory, checkOnly) {
		errors++
	}

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%s (%s):\n", color.Blu("Project memory"), color.Grn(hostname))
	if !setupLink(stdout, localProjects, cloudProjects, checkOnly) {
		errors++
	}

	fmt.Fprintln(stdout)
	if errors > 0 {
		if checkOnly {
			fmt.Fprintf(stdout, "CHECK: %d issue(s) found. Run 'claudecfg env' (without check) to fix.\n", errors)
		} else {
			fmt.Fprintf(stdout, "DONE: %d issue(s) could not be resolved automatically.\n", errors)
		}
		return 1
	}
	cFiles, cErrs := findConflicts(cloudMemory, cloudProjects)
	label := "DONE"
	if checkOnly {
		label = "CHECK"
	}
	var notes []string
	if len(cFiles) > 0 {
		notes = append(notes, fmt.Sprintf("%d unresolved conflict file(s) — run 'claudecfg env conflicts' to list", len(cFiles)))
	}
	if cErrs > 0 {
		notes = append(notes, "conflict scan incomplete — see warnings above")
	}
	if len(notes) > 0 {
		fmt.Fprintf(stdout, "%s: environment OK (%s)\n", label, strings.Join(notes, "; "))
	} else {
		fmt.Fprintf(stdout, "%s: environment OK\n", label)
	}
	return 0
}

func verifyCloudBase(w io.Writer) bool {
	target, err := os.Readlink(cloudBase)
	if err != nil {
		fmt.Fprintf(w, "FAIL: %s is not a symlink\n", cloudBase)
		return false
	}
	resolved := resolvePath(cloudBase, target)
	expected := filepath.Clean(filepath.Join(home, icloudTarget))
	if resolved != expected {
		fmt.Fprintf(w, "FAIL: %s resolves to '%s', expected '%s'\n", cloudBase, resolved, expected)
		return false
	}
	fmt.Fprintf(w, "  OK: %s -> '%s'\n", cloudBase, resolved)
	return true
}

func resolvePath(link, target string) string {
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	return filepath.Clean(target)
}

func setupLink(w io.Writer, localPath, cloudPath string, checkOnly bool) bool {
	fi, err := os.Stat(cloudPath)
	if err != nil || !fi.IsDir() {
		if checkOnly {
			fmt.Fprintf(w, "FAIL: %s does not exist\n", cloudPath)
			return false
		}
		if err := os.MkdirAll(cloudPath, 0755); err != nil {
			fmt.Fprintf(w, "FAIL: cannot create %s: %v\n", cloudPath, err)
			return false
		}
		fmt.Fprintf(w, "  CREATED: %s\n", cloudPath)
	} else {
		fmt.Fprintf(w, "  OK: %s exists\n", cloudPath)
	}

	lfi, lerr := os.Lstat(localPath)

	if lerr == nil && lfi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(localPath)
		if err != nil {
			fmt.Fprintf(w, "FAIL: cannot read symlink %s: %v\n", localPath, err)
			return false
		}
		if resolvePath(localPath, target) == filepath.Clean(cloudPath) {
			fmt.Fprintf(w, "  OK: %s -> %s\n", localPath, cloudPath)
			return true
		}
		fmt.Fprintf(w, "FAIL: %s is a symlink but points to '%s', expected '%s'\n", localPath, target, cloudPath)
		return false
	}

	if lerr == nil && !lfi.IsDir() {
		fmt.Fprintf(w, "FAIL: %s exists but is a %s, not a directory or symlink — remove it manually\n", localPath, lfi.Mode().Type())
		return false
	}

	if lerr == nil && lfi.IsDir() {
		if checkOnly {
			fmt.Fprintf(w, "FAIL: %s is a real directory (not symlinked)\n", localPath)
			return false
		}
		migrated, conflicts, err := migrateFiles(w, localPath, cloudPath)
		if err != nil {
			fmt.Fprintf(w, "FAIL: migration error: %v\n", err)
			return false
		}
		if migrated > 0 || conflicts > 0 {
			fmt.Fprintf(w, "  MIGRATED: %d files from %s to %s", migrated, localPath, cloudPath)
			if conflicts > 0 {
				fmt.Fprintf(w, " (%d conflict(s) preserved)", conflicts)
			}
			fmt.Fprintln(w)
		}
		if err := os.RemoveAll(localPath); err != nil {
			fmt.Fprintf(w, "FAIL: cannot remove %s: %v\n", localPath, err)
			return false
		}
		fmt.Fprintf(w, "  REMOVED: %s (contents migrated)\n", localPath)
	}

	if checkOnly {
		fmt.Fprintf(w, "FAIL: %s does not exist\n", localPath)
		return false
	}
	if err := os.Symlink(cloudPath, localPath); err != nil {
		fmt.Fprintf(w, "FAIL: cannot create symlink: %v\n", err)
		return false
	}
	fmt.Fprintf(w, "  LINKED: %s -> %s\n", localPath, cloudPath)
	return true
}

func migrateFiles(w io.Writer, src, dst string) (migrated, conflicts int, err error) {
	err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		if _, statErr := os.Stat(dstPath); statErr == nil {
			if filesEqual(path, dstPath) {
				return nil
			}
			conflictPath := nextConflictPath(dstPath)
			if cpErr := copyFile(path, conflictPath); cpErr != nil {
				return cpErr
			}
			conflicts++
			fmt.Fprintf(w, "  CONFLICT: %s saved as %s\n", rel, filepath.Base(conflictPath))
			return nil
		}
		if cpErr := copyFile(path, dstPath); cpErr != nil {
			return cpErr
		}
		migrated++
		return nil
	})
	return
}

func nextConflictPath(dstPath string) string {
	base := dstPath + ".conflict-" + hostname
	if _, err := os.Stat(base); err != nil {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s.%d", base, i)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}

func filesEqual(a, b string) bool {
	da, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(da, db)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

const conflictSuffix = ".conflict-"

func isConflictFile(name string) bool {
	idx := strings.LastIndex(name, conflictSuffix)
	if idx < 1 {
		return false
	}
	after := name[idx+len(conflictSuffix):]
	if before, after0, ok := strings.Cut(after, "."); ok {
		host := before
		num := after0
		if host == "" || num == "" {
			return false
		}
		for _, c := range num {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return after != ""
}

func findConflicts(dirs ...string) (found []string, scanErrors int) {
	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && isConflictFile(filepath.Base(path)) {
				found = append(found, path)
			}
			return nil
		})
		if err != nil {
			scanErrors++
		}
	}
	return
}

func runEnvConflicts(stdout io.Writer) int {
	cFiles, _ := findConflicts(cloudMemory, cloudProjects)
	if len(cFiles) == 0 {
		fmt.Fprintln(stdout, "No conflict files found.")
		return 0
	}
	fmt.Fprintf(stdout, "%d conflict file(s):\n\n", len(cFiles))
	for _, p := range cFiles {
		fmt.Fprintf(stdout, "  %s\n", p)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "To resolve: compare each conflict file with the original, keep the one you want, delete the other.")
	return 0
}

// --- perms subcommand ---

func runPerms(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Unknown command: (missing subcommand)")
		fmt.Fprint(stderr, permsHelp())
		return 1
	}
	switch args[0] {
	case "-h", "--help":
		fmt.Fprint(stdout, permsHelp())
		return 0
	case "init":
		return runPermsInit(args[1:], stdout, stderr)
	case "list":
		return runPermsList(stdout, stderr)
	case "show":
		return runPermsShow(args[1:], stdout, stderr)
	case "check":
		return runPermsCheck(args[1:], stdout, stderr)
	default:
		if strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "Unknown flag: %s\n", args[0])
		} else {
			fmt.Fprintf(stderr, "Unknown command: %s\n", args[0])
		}
		fmt.Fprint(stderr, permsHelp())
		return 1
	}
}

// --- perms init ---

func runPermsInit(args []string, stdout, stderr io.Writer) int {
	profileName := "go"
	dryRun := false
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-p", "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Unknown flag: -p requires a value")
				fmt.Fprint(stderr, permsHelp())
				return 1
			}
			profileName = args[i+1]
			i += 2
		case "-n", "--dry-run":
			dryRun = true
			i++
		case "-h", "--help":
			fmt.Fprint(stdout, permsHelp())
			return 0
		default:
			fmt.Fprintf(stderr, "Unknown flag: %s\n", args[i])
			fmt.Fprint(stderr, permsHelp())
			return 1
		}
	}

	profiles, _ := loadProfiles(stderr)
	profile, ok := profiles[profileName]
	if !ok {
		fmt.Fprintf(stderr, "ERROR: unknown profile '%s'\n", profileName)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot read cwd: %v\n", err)
		return 1
	}
	settingsDir := filepath.Join(cwd, ".claude")
	settingsPath := filepath.Join(settingsDir, "settings.json")

	merged, err := mergeSettings(settingsPath, profile)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	if dryRun {
		fmt.Fprintln(stdout, string(merged))
		return 0
	}

	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot create %s: %v\n", settingsDir, err)
		return 1
	}
	if err := os.WriteFile(settingsPath, merged, 0644); err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot write %s: %v\n", settingsPath, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", settingsPath)
	return 0
}

// profileData holds a parsed profile JSON. Only permissions.allow is
// deeply interpreted; other keys pass through as raw JSON.
type profileData struct {
	raw       []byte
	allowList []string
	// otherKeys preserves top-level keys other than "permissions".
	otherKeys map[string]json.RawMessage
	// permsOtherKeys preserves permissions.* keys other than "allow".
	permsOtherKeys map[string]json.RawMessage
}

// parseProfile parses a profile's raw JSON into allow list + other keys.
// Returns error if the JSON is invalid or missing permissions.
func parseProfile(data []byte) (*profileData, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	permsRaw, ok := top["permissions"]
	if !ok {
		return nil, fmt.Errorf("missing 'permissions' key")
	}
	var permsMap map[string]json.RawMessage
	if err := json.Unmarshal(permsRaw, &permsMap); err != nil {
		return nil, fmt.Errorf("invalid permissions object: %w", err)
	}
	var allow []string
	if allowRaw, ok := permsMap["allow"]; ok {
		if err := json.Unmarshal(allowRaw, &allow); err != nil {
			return nil, fmt.Errorf("invalid permissions.allow: %w", err)
		}
	}
	other := map[string]json.RawMessage{}
	for k, v := range top {
		if k != "permissions" {
			other[k] = v
		}
	}
	permsOther := map[string]json.RawMessage{}
	for k, v := range permsMap {
		if k != "allow" {
			permsOther[k] = v
		}
	}
	return &profileData{
		raw:            data,
		allowList:      allow,
		otherKeys:      other,
		permsOtherKeys: permsOther,
	}, nil
}

// mergeSettings reads an existing settings.json (if present), merges the
// profile's allow list as a deduplicated union (profile entries appended
// after existing), and returns the serialized merged JSON.
func mergeSettings(settingsPath string, profile *profileData) ([]byte, error) {
	var existing *profileData
	if data, err := os.ReadFile(settingsPath); err == nil {
		parsed, perr := parseProfile(data)
		if perr != nil {
			return nil, fmt.Errorf("existing %s: %w", settingsPath, perr)
		}
		existing = parsed
	}

	var allow []string
	seen := map[string]bool{}
	otherKeys := map[string]json.RawMessage{}
	permsOtherKeys := map[string]json.RawMessage{}

	if existing != nil {
		for _, e := range existing.allowList {
			if !seen[e] {
				allow = append(allow, e)
				seen[e] = true
			}
		}
		maps.Copy(otherKeys, existing.otherKeys)
		maps.Copy(permsOtherKeys, existing.permsOtherKeys)
	}
	for _, e := range profile.allowList {
		if !seen[e] {
			allow = append(allow, e)
			seen[e] = true
		}
	}

	// Rebuild permissions object: allow first, then preserved other keys.
	// json Marshal with map ordering is unstable; explicit ordering below.
	permsObj := orderedObject{}
	permsObj.add("allow", mustMarshal(allow))
	keys := make([]string, 0, len(permsOtherKeys))
	for k := range permsOtherKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		permsObj.add(k, permsOtherKeys[k])
	}

	topObj := orderedObject{}
	topObj.add("permissions", permsObj.marshal())
	keys = keys[:0]
	for k := range otherKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		topObj.add(k, otherKeys[k])
	}
	raw := topObj.marshal()

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return nil, err
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// orderedObject builds a JSON object with a deterministic key order.
type orderedObject struct {
	keys   []string
	values []json.RawMessage
}

func (o *orderedObject) add(k string, v json.RawMessage) {
	o.keys = append(o.keys, k)
	o.values = append(o.values, v)
}

func (o *orderedObject) marshal() json.RawMessage {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(o.values[i])
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// --- perms list / show ---

func runPermsList(stdout, stderr io.Writer) int {
	profiles, _ := loadProfiles(stderr)
	names := make([]string, 0, len(profiles))
	for n := range profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintln(stdout, n)
	}
	return 0
}

func runPermsShow(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "Usage: claudecfg perms show <profile>")
		return 1
	}
	name := args[0]
	profiles, _ := loadProfiles(stderr)
	p, ok := profiles[name]
	if !ok {
		fmt.Fprintf(stderr, "ERROR: unknown profile '%s'\n", name)
		return 1
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, p.raw, "", "  "); err != nil {
		fmt.Fprintf(stderr, "ERROR: malformed profile JSON: %v\n", err)
		return 1
	}
	pretty.WriteByte('\n')
	stdout.Write(pretty.Bytes())
	return 0
}

// loadProfiles returns the union of embedded + override-dir profiles.
// Override dir entries shadow embedded by filename stem. Malformed override
// files are skipped with a stderr WARNING; the embedded set is assumed valid.
func loadProfiles(stderr io.Writer) (map[string]*profileData, error) {
	out := map[string]*profileData{}

	entries, err := embeddedProfiles.ReadDir("profiles")
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, rerr := embeddedProfiles.ReadFile("profiles/" + e.Name())
			if rerr != nil {
				continue
			}
			parsed, perr := parseProfile(data)
			if perr != nil {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".json")
			out[name] = parsed
		}
	}

	if _, err := os.Stat(permsStarters); err == nil {
		dirEntries, rerr := os.ReadDir(permsStarters)
		if rerr == nil {
			for _, e := range dirEntries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
					continue
				}
				path := filepath.Join(permsStarters, e.Name())
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".json")
				parsed, perr := parseProfile(data)
				if perr != nil {
					fmt.Fprintf(stderr, "WARNING: skipping malformed override profile '%s' at %s: %v\n", name, path, perr)
					continue
				}
				out[name] = parsed
			}
		}
	}

	_ = fs.ErrNotExist
	return out, nil
}

// --- perms check ---

func runPermsCheck(args []string, stdout, stderr io.Writer) int {
	strict := false
	for _, a := range args {
		switch a {
		case "-s", "--strict":
			strict = true
		case "-h", "--help":
			fmt.Fprint(stdout, permsHelp())
			return 0
		default:
			fmt.Fprintf(stderr, "Unknown flag: %s\n", a)
			fmt.Fprint(stderr, permsHelp())
			return 1
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot read cwd: %v\n", err)
		return 1
	}

	files := []string{
		filepath.Join(cwd, ".claude", "settings.json"),
		filepath.Join(cwd, ".claude", "settings.local.json"),
	}

	totalWarns := 0
	malformed := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		warns := checkPermsFile(stdout, filepath.Base(f), data, true)
		if warns < 0 {
			malformed = true
			fmt.Fprintf(stderr, "ERROR: malformed JSON in %s\n", f)
			continue
		}
		totalWarns += warns
	}

	if malformed {
		return 1
	}
	if strict && totalWarns > 0 {
		return 1
	}
	return 0
}

// checkPermsFile writes WARN/INFO lines to w for each offending entry in
// the file's permissions.allow list. Returns the number of WARNs, or -1 if
// the JSON is malformed. If permsStatus is true, indent is used for the
// check subcommand; otherwise status-mode indent is used.
func checkPermsFile(w io.Writer, label string, data []byte, fromCheckCmd bool) int {
	parsed, err := parseProfile(data)
	if err != nil {
		if fromCheckCmd {
			return -1
		}
		return 0
	}
	warns := 0
	indent := "  "
	if fromCheckCmd {
		indent = ""
	}
	for _, entry := range parsed.allowList {
		for _, v := range analyzeEntry(entry) {
			fmt.Fprintf(w, "%s%s: %s %s — %s\n", indent, label, v.level, entry, v.reason)
			if v.level == "WARN" {
				warns++
			}
		}
	}
	return warns
}

type verdict struct {
	level  string // "WARN" or "INFO"
	reason string
}

var warnHeads = []string{
	"python", "python3", "node", "bun", "deno", "ruby", "perl", "php", "lua",
	"bash", "sh", "zsh", "fish", "eval", "exec", "ssh",
	"npx", "bunx", "uvx", "sudo",
	"npm run", "yarn run", "pnpm run", "bun run",
	"make", "just", "cargo run", "go run", "curl",
}

var infoHeads = []string{
	"ls", "cat", "grep", "rg", "echo", "printf", "cd", "wc", "head", "tail",
	"find", "which", "diff", "jq", "sed", "awk", "file", "strings",
	"readlink", "sleep", "test", "fd",
	"git status", "git log", "git diff", "git show", "git blame", "git branch", "git ls-files",
}

func analyzeEntry(entry string) []verdict {
	body, ok := extractBashBody(entry)
	if !ok {
		return nil
	}
	if body == "" {
		return nil
	}

	head := longestHeadMatch(body)
	wildcards := bodyWildcards(body, head)

	var out []verdict
	if head != "" && inList(head, warnHeads) && wildcards {
		out = append(out, verdict{level: "WARN", reason: fmt.Sprintf("arbitrary code execution via '%s'", head)})
	}
	if head != "" && inList(head, infoHeads) {
		out = append(out, verdict{level: "INFO", reason: fmt.Sprintf("'%s' is already auto-allowed by Claude Code", head)})
	}
	if containsAbsPath(body) && !isWildcardOverCommand(body) {
		out = append(out, verdict{level: "INFO", reason: "absolute path in pattern — likely a stale one-off"})
	}
	return out
}

// extractBashBody returns the inside of Bash(...) if the entry is shaped
// Bash(<body>). Returns ok=false for non-Bash entries, malformed shapes
// (missing closing paren), or empty Bash().
func extractBashBody(entry string) (string, bool) {
	const prefix = "Bash("
	if !strings.HasPrefix(entry, prefix) {
		return "", false
	}
	if !strings.HasSuffix(entry, ")") {
		return "", false
	}
	return entry[len(prefix) : len(entry)-1], true
}

// longestHeadMatch returns the longest candidate prefix of body (at
// space/colon boundaries) that appears in warnHeads ∪ infoHeads, or "" if
// no candidate matches.
func longestHeadMatch(body string) string {
	candidates := candidateHeads(body)
	longest := ""
	for _, c := range candidates {
		if (inList(c, warnHeads) || inList(c, infoHeads)) && len(c) > len(longest) {
			longest = c
		}
	}
	return longest
}

// candidateHeads returns successive prefixes of body at space/colon
// boundaries. E.g., "go list *" → ["go", "go list"], "python3:*" → ["python3"],
// "git status" → ["git", "git status"].
func candidateHeads(body string) []string {
	var cands []string
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == ' ' || c == ':' {
			cands = append(cands, body[:i])
		}
	}
	cands = append(cands, body) // whole body also a candidate
	// dedup while preserving order
	seen := map[string]bool{}
	out := cands[:0]
	for _, c := range cands {
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// bodyWildcards returns true if the body wildcards the command (contains
// any '*').
func bodyWildcards(body, head string) bool {
	return strings.Contains(body, "*")
}

// containsAbsPath returns true if the body contains a substring matching
// a known absolute-path prefix.
func containsAbsPath(body string) bool {
	for _, p := range []string{"/Users/", "/home/", "/tmp/"} {
		if strings.Contains(body, p) {
			return true
		}
	}
	return false
}

// isWildcardOverCommand returns true for bodies of shape <head>*, <head>:*,
// or <head> * — i.e., the wildcard is the only thing after the command.
func isWildcardOverCommand(body string) bool {
	for _, suffix := range []string{"*", ":*", " *"} {
		if before, ok := strings.CutSuffix(body, suffix); ok {
			rest := before
			if rest != "" && !strings.ContainsAny(rest, "*") {
				return true
			}
		}
	}
	return false
}

func inList(s string, list []string) bool {
	return slices.Contains(list, s)
}

// --- help text ---

func topLevelHelp() string {
	return `Usage: claudecfg [COMMAND] [FLAGS]

Default (no command): read-only status — renders env + perms sections.

Commands:
  env        Manage iCloud-backed Claude Code memory (mutating, macOS-only)
  perms      Manage project .claude/settings.json permissions

Flags:
  -v (--version)    Print version and exit
  -h (--help)       Print this help and exit

Run 'claudecfg env -h' or 'claudecfg perms -h' for subcommand details.
`
}

func envHelp() string {
	return `Usage: claudecfg env [SUBCOMMAND | FLAG]

Default (no further args): setup (mutating — creates symlinks, migrates files).
For read-only status, run 'claudecfg' with no arguments.

Subcommands:
  conflicts    List unresolved .conflict-<hostname> files under cloud dirs

Flags:
  -c (--check)     Verify-only mode — no mutation
  -i (--info)      Print design philosophy and how it works
  -h (--help)      Print this help
`
}

func permsHelp() string {
	return `Usage: claudecfg perms SUBCOMMAND [FLAGS]

Subcommands:
  init [-p NAME] [-n]    Seed .claude/settings.json from a profile (default: go)
  list                   List available profiles (alphabetical)
  show NAME              Print the named profile's JSON
  check [-s]             Audit the project's allowlist for risky patterns

Flags:
  -p (--profile) NAME    Profile name for init (default: go)
  -n (--dry-run)         Print intended writes without modifying (init)
  -s (--strict)          Exit 1 if any WARN found (check)
  -h (--help)            Print this help
`
}

func envInfoText() string {
	return `claudecfg env: Claude Code environment portability

PROBLEM
  Claude Code stores configuration across multiple layers, each with
  different portability characteristics:

  Layer 1: Repo governance (AGENTS.md, docs/role-*.md, etc.)
    - Lives in git, fully portable, agent-agnostic.
    - Any agent on any machine gets the same governance contract.
    - Managed by governa (or manually).

  Layer 2: Project-scoped memory (~/.claude/projects/<path>/memory/)
    - Machine-local, tied to a specific repo checkout path.
    - Contains corrections and context specific to a project.
    - Repo-germane rules should be migrated into Layer 1 governance.
    - Personal preferences and agent-specific workarounds stay here.

  Layer 3: Global memory (~/.claude/memory/)
    - Machine-local by default.
    - Contains personal interaction preferences that follow the user
      across all repos (response style, behavioral corrections, etc.).
    - THIS IS WHAT claudecfg env MAKES PORTABLE via iCloud symlink.

  Layer 4: Global settings (~/.claude/settings.json)
    - Machine-local. Contains plugins, permissions, effort level, voice.
    - Stays machine-specific (different machines may have different
      plugins or permissions). Small enough to recreate manually.

DESIGN PRINCIPLES
  - Repos are agent-agnostic: no Claude-specific config in repos.
  - Governance rules belong in governance docs, not agent memory.
  - Personal preferences follow the user, not the repo.
  - Machine-specific config stays machine-specific.
  - Simplest solution that works: symlink through iCloud, no tooling.

HOW IT WORKS
  ~/data/ is a symlink to iCloud CloudDrive on every machine.
  This tool creates two symlinks:

  1. ~/.claude/memory/ -> ~/data/etc/claude/memory/
     Global memory (Layer 3). Shared across all machines.

  2. ~/.claude/projects/ -> ~/data/etc/claude/<hostname>/projects/
     Project memory (Layer 2). Per-machine, but backed up to iCloud.
     Each machine gets its own namespace so checkout-path-specific
     memories don't collide across machines.

  Before:
    ~/.claude/memory/          (local directory, machine-bound)
    ~/.claude/projects/        (local directory, machine-bound)

  After:
    ~/.claude/memory/          (symlink -> ~/data/etc/claude/memory/)
    ~/.claude/projects/        (symlink -> ~/data/etc/claude/<hostname>/projects/)
    ~/data/etc/claude/memory/  (iCloud-synced, shared across machines)
    ~/data/etc/claude/<hostname>/projects/  (iCloud-synced, per-machine)

  On a new machine, run this tool once. If either directory already has
  files, they are migrated to the cloud directory first. Existing cloud
  files are never overwritten.

WHAT THIS DOES NOT DO
  - Does not sync settings.json (Layer 4). Machine-specific by design.
  - Does not modify any repo content or governance files.
  - Does not require any ongoing maintenance. Run once, forget.

RACE CONDITIONS
  If two machines write to the same memory file simultaneously, iCloud
  creates a "conflicted copy" file. This is rare in practice (you're
  typically on one machine at a time) and easily resolved manually.
`
}
