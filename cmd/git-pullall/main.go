package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ANSI color codes
const (
	Green   = "\033[1;32m"
	Red     = "\033[1;31m"
	Magenta = "\033[1;35m"
	Yellow  = "\033[1;33m"
	Blue    = "\033[1;34m"
	Reset   = "\033[0m"
)

const (
	program_name    = "git-pullall"
	program_version = "2.0.0"
)

func init() {
	_ = program_name
	_ = program_version
}

// isGitRepo checks if a directory is a git repository
func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// checkRemoteExists checks if the git remote repository is accessible
func checkRemoteExists(repoPath string) bool {
	cmd := exec.Command("git", "ls-remote")
	cmd.Dir = repoPath
	// Suppress output by not setting Stdout/Stderr
	err := cmd.Run()
	return err == nil
}

// pullRepo performs a git pull on the repository
func pullRepo(repoPath, repoName string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr for output
	output := stdout.String() + stderr.String()
	output = strings.TrimSpace(output)

	// Check if already up to date
	if strings.Contains(output, "Already up to date") || strings.Contains(output, "Already up-to-date") {
		fmt.Printf("==> %s%-35s%s %s\n", Yellow, repoName, Reset, "Already up to date")
		return nil
	}

	// Show repo name and full output for updates
	fmt.Printf("==> %s%-35s%s\n", Yellow, repoName, Reset)
	if output != "" {
		fmt.Println(output)
	}

	return err
}

// getDirectories returns all subdirectories in the current directory
func getDirectories() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func main() {
	// Get all directories in current directory
	dirs, err := getDirectories()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	if len(dirs) == 0 {
		fmt.Printf("%sNo directories found in current directory%s\n", Yellow, Reset)
		return
	}

	// Process each directory
	for _, dir := range dirs {
		// Check if it's a git repository
		if !isGitRepo(dir) {
			continue
		}

		// Check if remote exists
		if !checkRemoteExists(dir) {
			fmt.Printf("==> %s%-35s%s %s\n", Red, dir, Reset, "Repository not found, skipping...")
			continue
		}

		// Pull the repository
		if err := pullRepo(dir, dir); err != nil {
			fmt.Printf("==> %s%-35s%s %s\n", Red, dir, Reset, "Pull failed")
			// Continue to next repo instead of exiting
			continue
		}
	}
}
