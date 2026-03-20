package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRNHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_RN") != "1" {
		return
	}

	args := os.Args
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		os.Exit(2)
	}

	os.Args = append([]string{"rn"}, args[sep+1:]...)
	main()
	os.Exit(0)
}

func runRN(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestRNHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_RN=1")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestRNDryRunDoesNotRename(t *testing.T) {
	dir := t.TempDir()
	oldName := "report_old.txt"
	newName := "report_new.txt"

	if err := os.WriteFile(filepath.Join(dir, oldName), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := runRN(t, dir, "old", "new")
	if err != nil {
		t.Fatalf("dry-run command failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, oldName)); err != nil {
		t.Fatalf("expected original file to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, newName)); !os.IsNotExist(err) {
		t.Fatalf("expected renamed file to not exist in dry-run, stat err=%v", err)
	}
}

func TestRNForceRenamesFile(t *testing.T) {
	dir := t.TempDir()
	oldName := "report_old.txt"
	newName := "report_new.txt"

	if err := os.WriteFile(filepath.Join(dir, oldName), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := runRN(t, dir, "old", "new", "-f")
	if err != nil {
		t.Fatalf("force command failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, oldName)); !os.IsNotExist(err) {
		t.Fatalf("expected original file to be renamed away, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, newName)); err != nil {
		t.Fatalf("expected renamed file to exist: %v", err)
	}
}
