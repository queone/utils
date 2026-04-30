package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func withCleanHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	return home
}

func TestXDGStateDirHonorsAbsoluteEnvVar(t *testing.T) {
	home := withCleanHome(t)
	abs := filepath.Join(home, "custom-state")
	t.Setenv("XDG_STATE_HOME", abs)

	got, err := xdgStateDir()
	if err != nil {
		t.Fatalf("xdgStateDir: %v", err)
	}
	if got != abs {
		t.Errorf("xdgStateDir = %q, want %q", got, abs)
	}
}

func TestXDGStateDirIgnoresRelativeEnvVar(t *testing.T) {
	home := withCleanHome(t)
	t.Setenv("XDG_STATE_HOME", "relative/path")

	got, err := xdgStateDir()
	if err != nil {
		t.Fatalf("xdgStateDir: %v", err)
	}
	want := filepath.Join(home, ".local", "state")
	if got != want {
		t.Errorf("xdgStateDir = %q, want %q", got, want)
	}
}

func TestXDGStateDirFallsBackWhenUnset(t *testing.T) {
	home := withCleanHome(t)

	got, err := xdgStateDir()
	if err != nil {
		t.Fatalf("xdgStateDir: %v", err)
	}
	want := filepath.Join(home, ".local", "state")
	if got != want {
		t.Errorf("xdgStateDir = %q, want %q", got, want)
	}
}

func TestConfigPathResolvesNewLocation(t *testing.T) {
	home := withCleanHome(t)

	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath: %v", err)
	}
	want := filepath.Join(home, ".local", "state", "cash5", "draws.json")
	if got != want {
		t.Errorf("configPath = %q, want %q", got, want)
	}
}

func TestMigrationFromLegacyPath(t *testing.T) {
	home := withCleanHome(t)
	oldPath := filepath.Join(home, ".config", "cash5", "draws.json")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0755); err != nil {
		t.Fatalf("seed dir: %v", err)
	}
	body := []byte(`[{"id":"draw-1","gameName":"Cash 5","drawTime":1735689600000}]`)
	if err := os.WriteFile(oldPath, body, 0644); err != nil {
		t.Fatalf("seed old: %v", err)
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
		t.Errorf("old path still exists: err=%v", err)
	}

	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("content mismatch; got %q, want %q", got, body)
	}

	if !strings.Contains(stderr, "migrated") || !strings.Contains(stderr, oldPath) || !strings.Contains(stderr, newPath) {
		t.Errorf("stderr missing migration notice: %q", stderr)
	}
}

func TestMigrationSkippedWhenBothExist(t *testing.T) {
	home := withCleanHome(t)
	oldPath := filepath.Join(home, ".config", "cash5", "draws.json")
	newPath := filepath.Join(home, ".local", "state", "cash5", "draws.json")

	if err := os.MkdirAll(filepath.Dir(oldPath), 0755); err != nil {
		t.Fatalf("mkdir old: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		t.Fatalf("mkdir new: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("OLD"), 0644); err != nil {
		t.Fatalf("seed old: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("NEW"), 0644); err != nil {
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

	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new: %v", err)
	}
	if string(got) != "NEW" {
		t.Errorf("new file overwritten; got %q, want NEW", string(got))
	}
}

func TestMigrationSkippedForSymlink(t *testing.T) {
	home := withCleanHome(t)
	target := filepath.Join(home, "actual-draws.json")
	if err := os.WriteFile(target, []byte("[]"), 0644); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	oldPath := filepath.Join(home, ".config", "cash5", "draws.json")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
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

func TestColdStartReturnsEmpty(t *testing.T) {
	withCleanHome(t)

	stderr := captureStderr(t, func() {
		got, err := loadDraws()
		if err != nil {
			t.Fatalf("loadDraws: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d draws", len(got))
		}
	})

	if stderr != "" {
		t.Errorf("expected silent cold start, got stderr: %q", stderr)
	}
}

func TestSaveLoadRoundTripThroughResolver(t *testing.T) {
	withCleanHome(t)

	want := []Draw{
		{ID: "draw-1", DrawTime: 1735689600000, GameName: "Cash 5"},
		{ID: "draw-2", DrawTime: 1735776000000, GameName: "Cash 5"},
	}
	if err := saveDrawsCallback(want); err != nil {
		t.Fatalf("saveDrawsCallback: %v", err)
	}

	got, err := loadDraws()
	if err != nil {
		t.Fatalf("loadDraws: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	resolvedPath, err := configPath()
	if err != nil {
		t.Fatalf("configPath: %v", err)
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		t.Errorf("save did not land at configPath() result %s: %v", resolvedPath, err)
	}

	body, err := os.ReadFile(resolvedPath)
	if err != nil {
		t.Fatalf("read resolved: %v", err)
	}
	var roundTrip []Draw
	if err := json.Unmarshal(body, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(roundTrip) != len(want) {
		t.Errorf("round-trip mismatch: got %d, want %d", len(roundTrip), len(want))
	}
}
