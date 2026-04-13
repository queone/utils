package main

import (
	"bytes"
	"fmt"
	"github.com/queone/utils/internal/color"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	programName    = "claude-env"
	programVersion = "1.0.0"
)

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

var (
	home          = os.Getenv("HOME")
	hostname      = getHostname()
	icloudTarget  = "Library/Mobile Documents/com~apple~CloudDocs"
	cloudBase     = filepath.Join(home, "data")
	cloudClaude   = filepath.Join(cloudBase, "etc", "claude")
	cloudMemory   = filepath.Join(cloudClaude, "memory")
	cloudProjects = filepath.Join(cloudClaude, hostname, "projects")
	localClaude   = filepath.Join(home, ".claude")
	localMemory   = filepath.Join(localClaude, "memory")
	localProjects = filepath.Join(localClaude, "projects")
)

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "FAIL: claude-env requires macOS (uses iCloud for sync)")
		os.Exit(1)
	}

	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "info", "i":
		showInfo()
		return
	case "help", "h":
		showHelp()
		return
	case "version", "ver":
		fmt.Printf("%s v%s\n", programName, programVersion)
		return
	case "conflicts":
		showConflicts()
		return
	case "check", "c":
		run(true)
	case "":
		run(false)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", mode)
		showHelp()
		os.Exit(1)
	}
}

func run(checkOnly bool) {
	fmt.Println("claude-env: verifying Claude Code environment")
	fmt.Println()

	fmt.Println(color.Blu("iCloud base:"))
	if !verifyCloudBase() {
		fmt.Println()
		fmt.Println("Cannot proceed: ~/data must be a symlink to iCloud CloudDrive.")
		fmt.Printf("Expected: cd %s && ln -s '%s' data\n", home, icloudTarget)
		os.Exit(1)
	}

	if hostname == "" {
		fmt.Println()
		fmt.Println("FAIL: cannot determine hostname")
		os.Exit(1)
	}

	// Ensure ~/.claude/ exists
	if fi, err := os.Stat(localClaude); err != nil || !fi.IsDir() {
		if checkOnly {
			fmt.Printf("\nFAIL: %s does not exist\n", localClaude)
			os.Exit(1)
		}
		if err := os.MkdirAll(localClaude, 0755); err != nil {
			fmt.Printf("\nFAIL: cannot create %s: %v\n", localClaude, err)
			os.Exit(1)
		}
		fmt.Printf("\n  CREATED: %s\n", localClaude)
	}

	errors := 0

	fmt.Println()
	fmt.Println(color.Blu("Global memory:"))
	if !setupLink(localMemory, cloudMemory, checkOnly) {
		errors++
	}

	fmt.Println()
	fmt.Printf("%s (%s):\n", color.Blu("Project memory"), color.Grn(hostname))
	if !setupLink(localProjects, cloudProjects, checkOnly) {
		errors++
	}

	fmt.Println()
	if errors > 0 {
		if checkOnly {
			fmt.Printf("CHECK: %d issue(s) found. Run 'claude-env' (without check) to fix.\n", errors)
		} else {
			fmt.Printf("DONE: %d issue(s) could not be resolved automatically.\n", errors)
		}
		os.Exit(1)
	}
	cFiles, cErrs := findConflicts(cloudMemory, cloudProjects)
	label := "DONE"
	if checkOnly {
		label = "CHECK"
	}
	var notes []string
	if len(cFiles) > 0 {
		notes = append(notes, fmt.Sprintf("%d unresolved conflict file(s) — run 'claude-env conflicts' to list", len(cFiles)))
	}
	if cErrs > 0 {
		notes = append(notes, "conflict scan incomplete — see warnings above")
	}
	if len(notes) > 0 {
		fmt.Printf("%s: environment OK (%s)\n", label, strings.Join(notes, "; "))
	} else {
		fmt.Printf("%s: environment OK\n", label)
	}
}

func verifyCloudBase() bool {
	target, err := os.Readlink(cloudBase)
	if err != nil {
		fmt.Printf("FAIL: %s is not a symlink\n", cloudBase)
		return false
	}
	resolved := resolvePath(cloudBase, target)
	expected := filepath.Clean(filepath.Join(home, icloudTarget))
	if resolved != expected {
		fmt.Printf("FAIL: %s resolves to '%s', expected '%s'\n", cloudBase, resolved, expected)
		return false
	}
	fmt.Printf("  OK: %s -> '%s'\n", cloudBase, resolved)
	return true
}

// resolvePath resolves a symlink target relative to the symlink's parent
// directory and returns a cleaned absolute path.
func resolvePath(link, target string) string {
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	return filepath.Clean(target)
}

func setupLink(localPath, cloudPath string, checkOnly bool) bool {
	// Ensure cloud-side directory exists
	fi, err := os.Stat(cloudPath)
	if err != nil || !fi.IsDir() {
		if checkOnly {
			fmt.Printf("FAIL: %s does not exist\n", cloudPath)
			return false
		}
		if err := os.MkdirAll(cloudPath, 0755); err != nil {
			fmt.Printf("FAIL: cannot create %s: %v\n", cloudPath, err)
			return false
		}
		fmt.Printf("  CREATED: %s\n", cloudPath)
	} else {
		fmt.Printf("  OK: %s exists\n", cloudPath)
	}

	// Check current state of local path
	lfi, lerr := os.Lstat(localPath)

	// Case 1: already a symlink
	if lerr == nil && lfi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(localPath)
		if err != nil {
			fmt.Printf("FAIL: cannot read symlink %s: %v\n", localPath, err)
			return false
		}
		if resolvePath(localPath, target) == filepath.Clean(cloudPath) {
			fmt.Printf("  OK: %s -> %s\n", localPath, cloudPath)
			return true
		}
		fmt.Printf("FAIL: %s is a symlink but points to '%s', expected '%s'\n", localPath, target, cloudPath)
		return false
	}

	// Case 2: exists but is not a directory or symlink (e.g. regular file)
	if lerr == nil && !lfi.IsDir() {
		fmt.Printf("FAIL: %s exists but is a %s, not a directory or symlink — remove it manually\n", localPath, lfi.Mode().Type())
		return false
	}

	// Case 3: real directory — migrate contents then replace with symlink
	if lerr == nil && lfi.IsDir() {
		if checkOnly {
			fmt.Printf("FAIL: %s is a real directory (not symlinked)\n", localPath)
			return false
		}
		migrated, conflicts, err := migrateFiles(localPath, cloudPath)
		if err != nil {
			fmt.Printf("FAIL: migration error: %v\n", err)
			return false
		}
		if migrated > 0 || conflicts > 0 {
			fmt.Printf("  MIGRATED: %d files from %s to %s", migrated, localPath, cloudPath)
			if conflicts > 0 {
				fmt.Printf(" (%d conflict(s) preserved)", conflicts)
			}
			fmt.Println()
		}
		if err := os.RemoveAll(localPath); err != nil {
			fmt.Printf("FAIL: cannot remove %s: %v\n", localPath, err)
			return false
		}
		fmt.Printf("  REMOVED: %s (contents migrated)\n", localPath)
	}

	// Case 3: does not exist (or was just removed above) — create symlink
	if checkOnly {
		fmt.Printf("FAIL: %s does not exist\n", localPath)
		return false
	}
	if err := os.Symlink(cloudPath, localPath); err != nil {
		fmt.Printf("FAIL: cannot create symlink: %v\n", err)
		return false
	}
	fmt.Printf("  LINKED: %s -> %s\n", localPath, cloudPath)
	return true
}

// migrateFiles copies files from src to dst preserving directory structure.
// Files that already exist at the destination with identical content are
// skipped. Files that differ are preserved as <name>.conflict-<hostname>
// in the destination so no local data is lost.
// Returns the number of files migrated and the number of conflicts saved.
func migrateFiles(src, dst string) (migrated, conflicts int, err error) {
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

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// Check if destination already exists
		if _, statErr := os.Stat(dstPath); statErr == nil {
			if filesEqual(path, dstPath) {
				return nil // identical — safe to skip
			}
			// Content differs — save local version as conflict file
			conflictPath := nextConflictPath(dstPath)
			if cpErr := copyFile(path, conflictPath); cpErr != nil {
				return cpErr
			}
			conflicts++
			fmt.Printf("  CONFLICT: %s saved as %s\n", rel, filepath.Base(conflictPath))
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

// nextConflictPath returns a unique conflict filename by appending
// .conflict-<hostname> (or .conflict-<hostname>.2, .3, etc.) to avoid
// overwriting a previous conflict file.
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

// filesEqual returns true if both files exist and have identical content.
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

// conflictSuffix is the marker this tool uses for preserved conflict files.
const conflictSuffix = ".conflict-"

// isConflictFile returns true if name matches the pattern this tool creates:
// <original>.conflict-<hostname>[.<n>]
func isConflictFile(name string) bool {
	idx := strings.LastIndex(name, conflictSuffix)
	if idx < 1 { // must have a base name before the suffix
		return false
	}
	after := name[idx+len(conflictSuffix):]
	// after must be <hostname> or <hostname>.<digits>
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
	return after != "" // bare hostname, no dot
}

// findConflicts returns all conflict files under the given directories.
// If a directory cannot be walked, a warning is printed and scanning continues.
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
			fmt.Printf("  WARNING: could not fully scan %s: %v\n", dir, err)
			scanErrors++
		}
	}
	return
}

func showConflicts() {
	cFiles, _ := findConflicts(cloudMemory, cloudProjects)
	if len(cFiles) == 0 {
		fmt.Println("No conflict files found.")
		return
	}
	fmt.Printf("%d conflict file(s):\n\n", len(cFiles))
	for _, path := range cFiles {
		fmt.Printf("  %s\n", path)
	}
	fmt.Println()
	fmt.Println("To resolve: compare each conflict file with the original, keep the one you want, delete the other.")
}

func showHelp() {
	fmt.Printf("Usage: %s [command]\n", programName)
	fmt.Println()
	fmt.Println("  (no args)     Set up or repair the environment")
	fmt.Println("  check (c)     Verify-only mode (exit 0 if OK, 1 if not)")
	fmt.Println("  conflicts     List unresolved conflict files")
	fmt.Println("  info  (i)     Print design philosophy and how it works")
	fmt.Println("  version (ver) Show version")
	fmt.Println("  help  (h)     Show this help")
}

func showInfo() {
	fmt.Print(`claude-env: Claude Code environment portability

PROBLEM
  Claude Code stores configuration across multiple layers, each with
  different portability characteristics:

  Layer 1: Repo governance (AGENTS.md, docs/roles/, etc.)
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
    - THIS IS WHAT claude-env MAKES PORTABLE via iCloud symlink.

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
`)
}
