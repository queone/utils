package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr replaces os.Stderr with a pipe, runs fn, and returns
// whatever was written. Restores os.Stderr on return.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

// withCleanHome sets HOME and clears XDG_CONFIG_HOME so each test runs in a
// pristine temporary home with no carry-over.
func withCleanHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func TestXDGConfigDirHonorsAbsoluteEnvVar(t *testing.T) {
	home := withCleanHome(t)
	abs := filepath.Join(home, "custom-xdg")
	t.Setenv("XDG_CONFIG_HOME", abs)

	got, err := xdgConfigDir()
	if err != nil {
		t.Fatalf("xdgConfigDir: %v", err)
	}
	if got != abs {
		t.Errorf("xdgConfigDir = %q, want %q", got, abs)
	}
}

func TestXDGConfigDirIgnoresRelativeEnvVar(t *testing.T) {
	home := withCleanHome(t)
	t.Setenv("XDG_CONFIG_HOME", "relative/path")

	got, err := xdgConfigDir()
	if err != nil {
		t.Fatalf("xdgConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config")
	if got != want {
		t.Errorf("xdgConfigDir = %q, want %q (relative XDG_CONFIG_HOME should be ignored)", got, want)
	}
}

func TestXDGConfigDirFallsBackWhenUnset(t *testing.T) {
	home := withCleanHome(t)

	got, err := xdgConfigDir()
	if err != nil {
		t.Fatalf("xdgConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config")
	if got != want {
		t.Errorf("xdgConfigDir = %q, want %q", got, want)
	}
}

func TestConfigPathResolvesNewLocation(t *testing.T) {
	home := withCleanHome(t)

	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath: %v", err)
	}
	want := filepath.Join(home, ".config", "sms", "config.ini")
	if got != want {
		t.Errorf("configPath = %q, want %q", got, want)
	}
}

func TestMigrationFromLegacyPath(t *testing.T) {
	home := withCleanHome(t)
	oldPath := filepath.Join(home, ".smsrc")
	if err := os.WriteFile(oldPath, []byte("[global]\nsvckey = secret\n"), 0644); err != nil {
		t.Fatalf("seed old file: %v", err)
	}

	var newPath string
	stderr := captureStderr(t, func() {
		var err error
		newPath, err = configPath()
		if err != nil {
			t.Fatalf("configPath: %v", err)
		}
	})

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old path still exists after migration: err=%v", err)
	}

	info, err := os.Stat(newPath)
	if err != nil {
		t.Fatalf("new path missing after migration: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("new file mode = %v, want 0600", mode)
	}

	body, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new file: %v", err)
	}
	if !strings.Contains(string(body), "svckey = secret") {
		t.Errorf("migrated content lost; got %q", string(body))
	}

	if !strings.Contains(stderr, "migrated") || !strings.Contains(stderr, oldPath) || !strings.Contains(stderr, newPath) {
		t.Errorf("stderr missing migration notice: %q", stderr)
	}
}

func TestMigrationSkippedWhenBothExist(t *testing.T) {
	home := withCleanHome(t)
	oldPath := filepath.Join(home, ".smsrc")
	newPath := filepath.Join(home, ".config", "sms", "config.ini")

	if err := os.WriteFile(oldPath, []byte("OLD"), 0644); err != nil {
		t.Fatalf("seed old: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		t.Fatalf("mkdir new: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("NEW"), 0600); err != nil {
		t.Fatalf("seed new: %v", err)
	}

	stderr := captureStderr(t, func() {
		if _, err := configPath(); err != nil {
			t.Fatalf("configPath: %v", err)
		}
	})

	if !strings.Contains(stderr, "both") {
		t.Errorf("stderr missing both-exist warning: %q", stderr)
	}

	body, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new: %v", err)
	}
	if string(body) != "NEW" {
		t.Errorf("new file overwritten; got %q, want NEW", string(body))
	}

	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("old file deleted unexpectedly: %v", err)
	}
}

func TestMigrationSkippedForSymlink(t *testing.T) {
	home := withCleanHome(t)
	target := filepath.Join(home, "actual-config")
	if err := os.WriteFile(target, []byte("REAL"), 0600); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	oldPath := filepath.Join(home, ".smsrc")
	if err := os.Symlink(target, oldPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	stderr := captureStderr(t, func() {
		if _, err := configPath(); err != nil {
			t.Fatalf("configPath: %v", err)
		}
	})

	info, err := os.Lstat(oldPath)
	if err != nil {
		t.Fatalf("old symlink missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("old path is no longer a symlink (mode=%v)", info.Mode())
	}

	if !strings.Contains(stderr, "symlink") {
		t.Errorf("stderr missing symlink warning: %q", stderr)
	}
}

func TestColdStartNoFiles(t *testing.T) {
	withCleanHome(t)

	stderr := captureStderr(t, func() {
		if _, err := configPath(); err != nil {
			t.Fatalf("configPath: %v", err)
		}
	})

	if stderr != "" {
		t.Errorf("expected silent cold start, got stderr: %q", stderr)
	}
}

func TestCreateSkeletonWritesNewPath(t *testing.T) {
	home := withCleanHome(t)

	createSkeletonConfigFile()

	newPath := filepath.Join(home, ".config", "sms", "config.ini")
	info, err := os.Stat(newPath)
	if err != nil {
		t.Fatalf("expected skeleton at %s: %v", newPath, err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("skeleton mode = %v, want 0600", mode)
	}

	oldPath := filepath.Join(home, ".smsrc")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("legacy path %s should not exist after skeleton create", oldPath)
	}
}

func TestUsageTextMentionsNewPath(t *testing.T) {
	got := usageText()
	if !strings.Contains(got, "~/.config/sms/config.ini") {
		t.Errorf("usageText missing new path; got:\n%s", got)
	}
	if strings.Contains(got, "~/.smsrc") {
		t.Errorf("usageText still mentions legacy path; got:\n%s", got)
	}
}
