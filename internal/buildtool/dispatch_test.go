package buildtool

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Verify build.sh dispatches semver to cmd/rel, everything else to cmd/build.
func TestBuildShDispatch(t *testing.T) {
	content, err := os.ReadFile("../../build.sh")
	if err != nil {
		t.Fatalf("read build.sh: %v", err)
	}
	script := string(content)

	// Must contain semver detection that routes to cmd/rel
	semverPattern := regexp.MustCompile(`\$1.*=~.*\^v\[0-9\]`)
	if !semverPattern.MatchString(script) {
		t.Error("build.sh does not contain semver regex detection for first arg")
	}
	if !strings.Contains(script, "go run ./cmd/rel") {
		t.Error("build.sh does not dispatch to go run ./cmd/rel")
	}

	// Must contain fallthrough to cmd/build
	if !strings.Contains(script, "go run ./cmd/build") {
		t.Error("build.sh does not dispatch to go run ./cmd/build")
	}

	// Must use exec (replaces the shell process)
	lines := strings.Split(script, "\n")
	var execCount int
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "exec go run") {
			execCount++
		}
	}
	if execCount != 2 {
		t.Errorf("build.sh has %d 'exec go run' lines, want exactly 2 (one for rel, one for build)", execCount)
	}

	// The script should be a pure dispatcher — count non-boilerplate lines.
	var logicLines int
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "set ") {
			continue
		}
		if trimmed == "#!/usr/bin/env bash" || trimmed == "fi" {
			continue
		}
		logicLines++
	}
	if logicLines > 4 {
		t.Errorf("build.sh has %d logic lines, expected <=4 for a pure dispatcher", logicLines)
	}
}
