package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	program_name    = "git-cloneall"
	program_version = "2.0.0"
)

func init() {
	_ = program_name
	_ = program_version
}

// Repo represents a GitHub repository
type Repo struct {
	Name string `json:"name"`
}

// checkBinary checks if a binary exists in PATH
func checkBinary(name string) bool {
	fmt.Printf("==> Checking for %s%s%s ... ", Yellow, name, Reset)
	_, err := exec.LookPath(name)
	if err != nil {
		fmt.Printf("%smissing%s!\n", Red, Reset)
		return false
	}
	fmt.Printf("%sfound%s\n", Green, Reset)
	return true
}

// getRepoList fetches the list of repositories for an org/user using gh CLI
func getRepoList(orgUsername string) ([]string, error) {
	cmd := exec.Command("gh", "repo", "list", orgUsername, "--json", "name", "--jq", ".[].name")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	repos := strings.Split(output, "\n")
	return repos, nil
}

// cloneRepo clones a repository using git
func cloneRepo(orgUsername, repoName string) error {
	url := fmt.Sprintf("https://github.com/%s/%s.git", orgUsername, repoName)

	cmd := exec.Command("git", "clone", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// directoryExists checks if a directory exists
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// printUsage displays usage information
func printUsage() {
	fmt.Printf("Usage: %s <org/username>\n", program_name)
	fmt.Printf("\nClone all repositories from a GitHub organization or user.\n")
	fmt.Printf("\nExample:\n")
	fmt.Printf("  %s myorg\n", program_name)
	fmt.Printf("  %s someuser\n", program_name)
}

func main() {
	// Check for argument
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	orgUsername := os.Args[1]

	// Confirm required binaries are available
	allFound := true
	for _, binary := range []string{"git", "gh"} {
		if !checkBinary(binary) {
			allFound = false
		}
	}

	if !allFound {
		fmt.Fprintf(os.Stderr, "\n%sError: Required binaries are missing%s\n", Red, Reset)
		fmt.Println("\nTo install gh CLI:")
		fmt.Println("  brew install gh        # macOS")
		fmt.Println("  See: https://cli.github.com/")
		os.Exit(1)
	}

	fmt.Println()

	// Get list of repositories
	fmt.Printf("==> Fetching repositories for %s%s%s...\n", Yellow, orgUsername, Reset)
	repos, err := getRepoList(orgUsername)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	if len(repos) == 0 {
		fmt.Printf("%sNo repositories found for '%s'%s\n", Yellow, orgUsername, Reset)
		return
	}

	fmt.Printf("==> Found %s%d%s repositories\n\n", Green, len(repos), Reset)

	// Clone each repository
	clonedCount := 0
	skippedCount := 0

	for _, repo := range repos {
		// Check if directory already exists
		if directoryExists(repo) {
			fmt.Printf("==> %sWarning: Directory '%s' already exists. Skipping...%s\n", Yellow, repo, Reset)
			skippedCount++
			continue
		}

		fmt.Printf("==> %sCloning '%s'...%s\n", Yellow, repo, Reset)
		if err := cloneRepo(orgUsername, repo); err != nil {
			fmt.Fprintf(os.Stderr, "%sError cloning '%s': %v%s\n", Red, repo, err, Reset)
			continue
		}
		clonedCount++
		fmt.Println()
	}

	// Summary
	fmt.Printf("\n%s✓ Summary:%s\n", Green, Reset)
	fmt.Printf("  Cloned: %d\n", clonedCount)
	fmt.Printf("  Skipped: %d\n", skippedCount)
	fmt.Printf("  Total: %d\n", len(repos))
}
