package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFRHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_FR") != "1" {
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

	os.Args = append([]string{"fr"}, args[sep+1:]...)
	main()
	os.Exit(0)
}

func runFR(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestFRHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_FR=1")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCountMatches(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		pattern string
		want    int
	}{
		{name: "two matches", data: []byte("foo bar foo"), pattern: "foo", want: 2},
		{name: "regex match", data: []byte("a1 a2 a3"), pattern: `a[0-9]`, want: 3},
		{name: "invalid regex", data: []byte("abc"), pattern: "(", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countMatches(tc.data, tc.pattern)
			if got != tc.want {
				t.Fatalf("countMatches() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReplaceAll(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		pattern string
		to      string
		want    string
	}{
		{name: "replace literal", data: []byte("foo bar foo"), pattern: "foo", to: "baz", want: "baz bar baz"},
		{name: "replace regex", data: []byte("a1 a2"), pattern: `a[0-9]`, to: "x", want: "x x"},
		{name: "invalid regex unchanged", data: []byte("abc"), pattern: "(", to: "x", want: "abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(replaceAll(tc.data, tc.pattern, tc.to))
			if got != tc.want {
				t.Fatalf("replaceAll() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFRReplaceModeUpdatesTextFiles(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "note.txt")
	binaryPath := filepath.Join(dir, "blob.bin")
	origBinary := []byte{0x00, 0xff, 0x00, 0x10}

	if err := os.WriteFile(textPath, []byte("foo\nfoo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, origBinary, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := runFR(t, dir, "foo", "bar", "-f")
	if err != nil {
		t.Fatalf("replace command failed: %v", err)
	}

	textData, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(textData); got != "bar\nbar\n" {
		t.Fatalf("text file replacement mismatch: got %q", got)
	}

	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(binaryData) != len(origBinary) {
		t.Fatalf("binary file size changed unexpectedly: got=%d want=%d", len(binaryData), len(origBinary))
	}
	for i := range origBinary {
		if binaryData[i] != origBinary[i] {
			t.Fatalf("binary file content changed at byte %d: got=%v want=%v", i, binaryData, origBinary)
		}
	}
}
