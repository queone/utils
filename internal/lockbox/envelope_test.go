package lockbox

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// testKDF keeps passphrase derivation fast in tests.
var testKDF = KDF{Time: 1, Memory: 8 * 1024, Threads: 1}

func newTestHeader(t *testing.T, key []byte, passphrase string) Header {
	t.Helper()
	id, err := NewKeyID()
	if err != nil {
		t.Fatal(err)
	}
	h := Header{Generation: 1, KeyID: id}
	if err := h.WrapKey(key, []byte(passphrase), testKDF); err != nil {
		t.Fatal(err)
	}
	return h
}

func mustKey(t *testing.T) []byte {
	t.Helper()
	k, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestSealOpenRoundTripAndEveryByteFlipFails(t *testing.T) {
	key := mustKey(t)
	h := newTestHeader(t, key, "pw")
	plain := []byte("sqlite bytes stand-in")
	file, err := Seal(h, key, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, out, err := Open(file, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("round trip mismatch: %q", out)
	}
	if got.Generation != 1 || got.KeyID != h.KeyID || got.Wrapped != h.Wrapped {
		t.Fatalf("header fields did not survive the round trip")
	}
	for i := range file {
		bad := bytes.Clone(file)
		bad[i] ^= 0xff
		if _, _, err := Open(bad, key); err == nil {
			t.Fatalf("byte %d flipped but open succeeded", i)
		}
	}
}

func TestKeyRecoveryWithPassphrase(t *testing.T) {
	key := mustKey(t)
	h := newTestHeader(t, key, "correct horse")
	file, err := Seal(h, key, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseHeader(file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parsed.UnwrapKey([]byte("correct horse"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, key) {
		t.Fatal("unwrapped key differs from the data key")
	}
	if _, err := parsed.UnwrapKey([]byte("wrong")); !errors.Is(err, ErrPassphrase) {
		t.Fatalf("wrong passphrase: got %v, want ErrPassphrase", err)
	}
	other := mustKey(t)
	if _, _, err := Open(file, other); !errors.Is(err, ErrAuth) {
		t.Fatalf("wrong data key: got %v, want ErrAuth", err)
	}
}

func TestParseHeaderRejectsForeignFiles(t *testing.T) {
	if _, err := ParseHeader([]byte("hello")); !errors.Is(err, ErrFormat) {
		t.Fatalf("short file: got %v", err)
	}
	junk := bytes.Repeat([]byte("Z"), headerSize+10)
	if _, err := ParseHeader(junk); !errors.Is(err, ErrFormat) {
		t.Fatalf("junk file: got %v", err)
	}
}

func TestWriteAtomicGenerationConflictAndMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.store")
	key := mustKey(t)
	h := newTestHeader(t, key, "pw")
	h.Generation = 1
	f1, _ := Seal(h, key, []byte("one"))
	if err := WriteAtomic(path, f1, 0); err != nil {
		t.Fatal(err)
	}
	h.Generation = 2
	f2, _ := Seal(h, key, []byte("two"))
	if err := WriteAtomic(path, f2, 1); err != nil {
		t.Fatal(err)
	}
	stale, _ := Seal(h, key, []byte("stale"))
	if err := WriteAtomic(path, stale, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale writer: got %v, want ErrConflict", err)
	}
	cur, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cur, f2) {
		t.Fatal("newer store was overwritten by a stale writer")
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("store mode %o, want 600", st.Mode().Perm())
	}
	if err := WriteAtomic(path, f1, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("init over existing store: got %v, want ErrConflict", err)
	}
	names := dirNames(t, dir)
	if len(names) != 1 || names[0] != "s.store" {
		t.Fatalf("directory holds %v, want only s.store", names)
	}
}

func TestWriteAtomicRenameFailureLeavesPreviousStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.store")
	key := mustKey(t)
	h := newTestHeader(t, key, "pw")
	f1, _ := Seal(h, key, []byte("one"))
	if err := WriteAtomic(path, f1, 0); err != nil {
		t.Fatal(err)
	}
	orig := renameFile
	renameFile = func(string, string) error { return errors.New("injected rename failure") }
	defer func() { renameFile = orig }()
	h.Generation = 2
	f2, _ := Seal(h, key, []byte("two"))
	if err := WriteAtomic(path, f2, 1); err == nil {
		t.Fatal("rename failure was not reported")
	}
	cur, _ := os.ReadFile(path)
	if !bytes.Equal(cur, f1) {
		t.Fatal("previous store changed after a failed rename")
	}
	names := dirNames(t, dir)
	if len(names) != 1 || names[0] != "s.store" {
		t.Fatalf("directory holds %v after failed rename, want only s.store", names)
	}
}

func TestConflictCopies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "macfit.store")
	for _, n := range []string{"macfit.store", "macfit 2.store", "macfit 13.store", "macfit x.store", "other.store", "macfit 2.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := ConflictCopies(path)
	want := []string{filepath.Join(dir, "macfit 13.store"), filepath.Join(dir, "macfit 2.store")}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("conflict copies %v, want %v", got, want)
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
