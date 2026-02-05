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
	program_name    = "git-remotev"
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

// getRemoteURL returns the origin remote URL for a git repository
func getRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "<no origin>"
	}

	return strings.TrimSpace(stdout.String())
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

		// Get and display remote URL
		remoteURL := getRemoteURL(dir)
		fmt.Printf("==> %s%-35s%s %s\n", Yellow, dir, Reset, remoteURL)
	}
}
