package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyCloudBase(t *testing.T) {
	tmp := t.TempDir()

	origHome := home
	origCloudBase := cloudBase
	origIcloudTarget := icloudTarget
	defer func() {
		home = origHome
		cloudBase = origCloudBase
		icloudTarget = origIcloudTarget
	}()

	home = tmp
	icloudTarget = "Library/Mobile Documents/com~apple~CloudDocs"
	cloudBase = filepath.Join(tmp, "data")

	t.Run("not_a_symlink", func(t *testing.T) {
		os.MkdirAll(cloudBase, 0755)
		defer os.RemoveAll(cloudBase)
		if verifyCloudBase() {
			t.Error("expected failure for real directory")
		}
	})

	t.Run("wrong_target", func(t *testing.T) {
		os.Remove(cloudBase)
		os.Symlink("wrong/path", cloudBase)
		defer os.Remove(cloudBase)
		if verifyCloudBase() {
			t.Error("expected failure for wrong symlink target")
		}
	})

	t.Run("correct_relative_symlink", func(t *testing.T) {
		os.Remove(cloudBase)
		os.Symlink(icloudTarget, cloudBase)
		defer os.Remove(cloudBase)
		if !verifyCloudBase() {
			t.Error("expected success for correct relative symlink")
		}
	})

	t.Run("correct_absolute_symlink", func(t *testing.T) {
		os.Remove(cloudBase)
		os.Symlink(filepath.Join(home, icloudTarget), cloudBase)
		defer os.Remove(cloudBase)
		if !verifyCloudBase() {
			t.Error("expected success for correct absolute symlink")
		}
	})

	t.Run("correct_dot_relative_symlink", func(t *testing.T) {
		os.Remove(cloudBase)
		os.Symlink("./"+icloudTarget, cloudBase)
		defer os.Remove(cloudBase)
		if !verifyCloudBase() {
			t.Error("expected success for ./-prefixed relative symlink")
		}
	})
}

func TestSetupLink(t *testing.T) {
	t.Run("creates_both_dirs_and_symlink", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(filepath.Dir(local), 0755)

		if !setupLink(local, cloud, false) {
			t.Fatal("setupLink returned false")
		}

		fi, err := os.Stat(cloud)
		if err != nil || !fi.IsDir() {
			t.Error("cloud directory not created")
		}

		target, err := os.Readlink(local)
		if err != nil {
			t.Fatalf("local is not a symlink: %v", err)
		}
		if target != cloud {
			t.Errorf("symlink target = %q, want %q", target, cloud)
		}
	})

	t.Run("check_mode_missing_cloud", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		if setupLink(local, cloud, true) {
			t.Error("expected failure in check mode with missing cloud dir")
		}
	})

	t.Run("already_correct_symlink", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(cloud, 0755)
		os.MkdirAll(filepath.Dir(local), 0755)
		os.Symlink(cloud, local)

		if !setupLink(local, cloud, false) {
			t.Error("expected success for already-correct symlink")
		}
		if !setupLink(local, cloud, true) {
			t.Error("expected success for check mode on correct symlink")
		}
	})

	t.Run("correct_relative_symlink", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(cloud, 0755)
		os.MkdirAll(filepath.Dir(local), 0755)
		// Create a relative symlink that resolves to the same path
		rel, _ := filepath.Rel(filepath.Dir(local), cloud)
		os.Symlink(rel, local)

		if !setupLink(local, cloud, false) {
			t.Error("expected success for equivalent relative symlink")
		}
	})

	t.Run("wrong_symlink_target", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(cloud, 0755)
		os.MkdirAll(filepath.Dir(local), 0755)
		os.Symlink("/wrong/target", local)

		if setupLink(local, cloud, false) {
			t.Error("expected failure for wrong symlink target")
		}
	})

	t.Run("regular_file_blocking", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(cloud, 0755)
		os.MkdirAll(filepath.Dir(local), 0755)
		os.WriteFile(local, []byte("blocker"), 0644)

		// Capture stdout to verify guidance message
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := setupLink(local, cloud, false)

		w.Close()
		os.Stdout = oldStdout
		buf, _ := io.ReadAll(r)
		output := string(buf)

		if result {
			t.Error("expected failure when regular file blocks path")
		}
		if !strings.Contains(output, "remove it manually") {
			t.Errorf("output missing 'remove it manually' guidance: %q", output)
		}
	})

	t.Run("migrates_existing_directory", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		os.MkdirAll(filepath.Join(local, "sub"), 0755)
		os.WriteFile(filepath.Join(local, "a.md"), []byte("aaa"), 0644)
		os.WriteFile(filepath.Join(local, "sub", "b.md"), []byte("bbb"), 0644)

		if !setupLink(local, cloud, false) {
			t.Fatal("setupLink returned false")
		}

		if data, err := os.ReadFile(filepath.Join(cloud, "a.md")); err != nil || string(data) != "aaa" {
			t.Error("a.md not migrated correctly")
		}
		if data, err := os.ReadFile(filepath.Join(cloud, "sub", "b.md")); err != nil || string(data) != "bbb" {
			t.Error("sub/b.md not migrated correctly")
		}

		target, err := os.Readlink(local)
		if err != nil {
			t.Fatalf("local is not a symlink: %v", err)
		}
		if target != cloud {
			t.Errorf("symlink target = %q, want %q", target, cloud)
		}
	})

	t.Run("migration_preserves_conflicting_local_files", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		origHostname := hostname
		hostname = "testhost"
		defer func() { hostname = origHostname }()

		// Cloud already has a file
		os.MkdirAll(cloud, 0755)
		os.WriteFile(filepath.Join(cloud, "a.md"), []byte("cloud-version"), 0644)

		// Local has the same file with different content
		os.MkdirAll(local, 0755)
		os.WriteFile(filepath.Join(local, "a.md"), []byte("local-version"), 0644)

		if !setupLink(local, cloud, false) {
			t.Fatal("setupLink returned false")
		}

		// Cloud file should be preserved (not overwritten)
		data, _ := os.ReadFile(filepath.Join(cloud, "a.md"))
		if string(data) != "cloud-version" {
			t.Errorf("cloud file overwritten: got %q, want %q", data, "cloud-version")
		}

		// Local version should be saved as conflict file
		conflictData, err := os.ReadFile(filepath.Join(cloud, "a.md.conflict-testhost"))
		if err != nil {
			t.Fatalf("conflict file not created: %v", err)
		}
		if string(conflictData) != "local-version" {
			t.Errorf("conflict file content = %q, want %q", conflictData, "local-version")
		}
	})

	t.Run("migration_skips_identical_files", func(t *testing.T) {
		tmp := t.TempDir()
		local := filepath.Join(tmp, "local", "memory")
		cloud := filepath.Join(tmp, "cloud", "memory")

		// Both sides have the same file with identical content
		os.MkdirAll(cloud, 0755)
		os.WriteFile(filepath.Join(cloud, "a.md"), []byte("same"), 0644)
		os.MkdirAll(local, 0755)
		os.WriteFile(filepath.Join(local, "a.md"), []byte("same"), 0644)

		if !setupLink(local, cloud, false) {
			t.Fatal("setupLink returned false")
		}

		// No conflict file should exist
		_, err := os.Stat(filepath.Join(cloud, "a.md.conflict-"+hostname))
		if err == nil {
			t.Error("conflict file created for identical content")
		}
	})
}

func TestMigrateFiles(t *testing.T) {
	t.Run("empty_source", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		dst := filepath.Join(tmp, "dst")
		os.MkdirAll(src, 0755)
		os.MkdirAll(dst, 0755)

		migrated, conflicts, err := migrateFiles(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if migrated != 0 {
			t.Errorf("migrated = %d, want 0", migrated)
		}
		if conflicts != 0 {
			t.Errorf("conflicts = %d, want 0", conflicts)
		}
	})

	t.Run("conflict_preserved", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		dst := filepath.Join(tmp, "dst")
		os.MkdirAll(src, 0755)
		os.MkdirAll(dst, 0755)

		origHostname := hostname
		hostname = "testhost"
		defer func() { hostname = origHostname }()

		os.WriteFile(filepath.Join(src, "x.md"), []byte("new"), 0644)
		os.WriteFile(filepath.Join(dst, "x.md"), []byte("existing"), 0644)

		migrated, conflicts, err := migrateFiles(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if migrated != 0 {
			t.Errorf("migrated = %d, want 0", migrated)
		}
		if conflicts != 1 {
			t.Errorf("conflicts = %d, want 1", conflicts)
		}

		// Destination should be unchanged
		data, _ := os.ReadFile(filepath.Join(dst, "x.md"))
		if string(data) != "existing" {
			t.Errorf("existing file overwritten: got %q", data)
		}

		// Conflict file should contain local version
		conflictData, _ := os.ReadFile(filepath.Join(dst, "x.md.conflict-testhost"))
		if string(conflictData) != "new" {
			t.Errorf("conflict file content = %q, want %q", conflictData, "new")
		}
	})

	t.Run("identical_skipped_no_conflict", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		dst := filepath.Join(tmp, "dst")
		os.MkdirAll(src, 0755)
		os.MkdirAll(dst, 0755)

		os.WriteFile(filepath.Join(src, "x.md"), []byte("same"), 0644)
		os.WriteFile(filepath.Join(dst, "x.md"), []byte("same"), 0644)

		migrated, conflicts, err := migrateFiles(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if migrated != 0 {
			t.Errorf("migrated = %d, want 0", migrated)
		}
		if conflicts != 0 {
			t.Errorf("conflicts = %d, want 0", conflicts)
		}
	})

	t.Run("repeated_conflicts_get_numbered", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src")
		dst := filepath.Join(tmp, "dst")
		os.MkdirAll(src, 0755)
		os.MkdirAll(dst, 0755)

		origHostname := hostname
		hostname = "testhost"
		defer func() { hostname = origHostname }()

		// Existing cloud file
		os.WriteFile(filepath.Join(dst, "x.md"), []byte("cloud"), 0644)
		// Previous conflict file from an earlier run
		os.WriteFile(filepath.Join(dst, "x.md.conflict-testhost"), []byte("old-local"), 0644)

		// New local version differs from cloud
		os.WriteFile(filepath.Join(src, "x.md"), []byte("new-local"), 0644)

		_, conflicts, err := migrateFiles(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if conflicts != 1 {
			t.Errorf("conflicts = %d, want 1", conflicts)
		}

		// Old conflict should be untouched
		data, _ := os.ReadFile(filepath.Join(dst, "x.md.conflict-testhost"))
		if string(data) != "old-local" {
			t.Errorf("old conflict overwritten: got %q", data)
		}

		// New conflict should be numbered .2
		data2, err := os.ReadFile(filepath.Join(dst, "x.md.conflict-testhost.2"))
		if err != nil {
			t.Fatalf("numbered conflict file not created: %v", err)
		}
		if string(data2) != "new-local" {
			t.Errorf("numbered conflict content = %q, want %q", data2, "new-local")
		}
	})
}

func TestIsConflictFile(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"a.md.conflict-host1", true},
		{"a.md.conflict-host1.2", true},
		{"a.md.conflict-host1.37", true},
		{"x.conflict-np10", true},
		{".conflict-host", false},         // no base name before suffix
		{"a.md.conflict-", false},         // empty hostname
		{"a.md.conflict-host.abc", false}, // non-numeric counter
		{"a.md.conflict-host.", false},    // trailing dot, empty counter
		{"a.md", false},
		{"readme.txt", false},
		{"some.conflict-in-name.md", false}, // not at expected position pattern
	}
	for _, tc := range cases {
		got := isConflictFile(tc.name)
		if got != tc.want {
			t.Errorf("isConflictFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestFindConflicts(t *testing.T) {
	t.Run("no_conflicts", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.md"), []byte("ok"), 0644)

		found, errs := findConflicts(tmp)
		if len(found) != 0 {
			t.Errorf("found %d conflicts, want 0", len(found))
		}
		if errs != 0 {
			t.Errorf("errs = %d, want 0", errs)
		}
	})

	t.Run("finds_conflict_files", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "a.md"), []byte("ok"), 0644)
		os.WriteFile(filepath.Join(tmp, "a.md.conflict-host1"), []byte("c1"), 0644)
		os.WriteFile(filepath.Join(tmp, "a.md.conflict-host1.2"), []byte("c2"), 0644)
		os.MkdirAll(filepath.Join(tmp, "sub"), 0755)
		os.WriteFile(filepath.Join(tmp, "sub", "b.md.conflict-host2"), []byte("c3"), 0644)

		found, errs := findConflicts(tmp)
		if len(found) != 3 {
			t.Errorf("found %d conflicts, want 3", len(found))
		}
		if errs != 0 {
			t.Errorf("errs = %d, want 0", errs)
		}
	})

	t.Run("ignores_false_positives", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "some.conflict-in-name.md"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(tmp, ".conflict-host"), []byte("x"), 0644)

		found, _ := findConflicts(tmp)
		if len(found) != 0 {
			t.Errorf("found %d conflicts, want 0 (false positives)", len(found))
		}
	})

	t.Run("scans_multiple_dirs", func(t *testing.T) {
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "dir1")
		dir2 := filepath.Join(tmp, "dir2")
		os.MkdirAll(dir1, 0755)
		os.MkdirAll(dir2, 0755)
		os.WriteFile(filepath.Join(dir1, "x.conflict-a"), []byte("c"), 0644)
		os.WriteFile(filepath.Join(dir2, "y.conflict-b"), []byte("c"), 0644)

		found, errs := findConflicts(dir1, dir2)
		if len(found) != 2 {
			t.Errorf("found %d conflicts, want 2", len(found))
		}
		if errs != 0 {
			t.Errorf("errs = %d, want 0", errs)
		}
	})

	t.Run("missing_dir_reports_error", func(t *testing.T) {
		found, errs := findConflicts("/nonexistent/path")
		if len(found) != 0 {
			t.Errorf("found %d conflicts, want 0", len(found))
		}
		if errs != 1 {
			t.Errorf("errs = %d, want 1", errs)
		}
	})
}

func TestShowConflicts(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		tmp := t.TempDir()
		origCloudMemory := cloudMemory
		origCloudProjects := cloudProjects
		cloudMemory = filepath.Join(tmp, "memory")
		cloudProjects = filepath.Join(tmp, "projects")
		defer func() { cloudMemory = origCloudMemory; cloudProjects = origCloudProjects }()

		os.MkdirAll(cloudMemory, 0755)
		os.MkdirAll(cloudProjects, 0755)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		showConflicts()
		w.Close()
		os.Stdout = oldStdout
		buf, _ := io.ReadAll(r)
		output := string(buf)

		if !strings.Contains(output, "No conflict files") {
			t.Errorf("unexpected output: %q", output)
		}
	})

	t.Run("with_conflicts", func(t *testing.T) {
		tmp := t.TempDir()
		origCloudMemory := cloudMemory
		origCloudProjects := cloudProjects
		cloudMemory = filepath.Join(tmp, "memory")
		cloudProjects = filepath.Join(tmp, "projects")
		defer func() { cloudMemory = origCloudMemory; cloudProjects = origCloudProjects }()

		os.MkdirAll(cloudMemory, 0755)
		os.MkdirAll(cloudProjects, 0755)
		os.WriteFile(filepath.Join(cloudMemory, "a.md.conflict-host"), []byte("c"), 0644)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		showConflicts()
		w.Close()
		os.Stdout = oldStdout
		buf, _ := io.ReadAll(r)
		output := string(buf)

		if !strings.Contains(output, "1 conflict file(s)") {
			t.Errorf("unexpected output: %q", output)
		}
		if !strings.Contains(output, "a.md.conflict-host") {
			t.Errorf("missing conflict path in output: %q", output)
		}
		if !strings.Contains(output, "resolve") {
			t.Errorf("missing resolution guidance in output: %q", output)
		}
	})
}
