package lockbox

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T, dir string) (*Store, []byte) {
	t.Helper()
	key := mustKey(t)
	h := newTestHeader(t, key, "pw")
	h.Generation = 0
	st, err := Create(filepath.Join(dir, "macfit.store"), h, key)
	if err != nil {
		t.Fatal(err)
	}
	return st, key
}

func TestStoreLifecycleWritesOnlyTheStoreFile(t *testing.T) {
	dir := t.TempDir()
	st, key := newTestStore(t, dir)
	e, err := st.AddEntry("~/.bashrc", 0o644, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddVersion(e.ID, []byte("export X=1\n"), "np10"); err != nil {
		t.Fatal(err)
	}
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	if st.Header.Generation != 1 {
		t.Fatalf("generation after first save %d, want 1", st.Header.Generation)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	names := dirNames(t, dir)
	if len(names) != 1 || names[0] != "macfit.store" {
		t.Fatalf("directory holds %v, want only macfit.store", names)
	}
	info, err := os.Stat(filepath.Join(dir, "macfit.store"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store mode %o, want 600", info.Mode().Perm())
	}

	re, err := Load(filepath.Join(dir, "macfit.store"), key)
	if err != nil {
		t.Fatal(err)
	}
	defer re.Close()
	entries, err := re.Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Target != "~/.bashrc" || entries[0].Mode != 0o644 || entries[0].Host != "" {
		t.Fatalf("entries after reload: %+v", entries)
	}
	v, ok, err := re.Latest(entries[0].ID)
	if err != nil || !ok {
		t.Fatalf("latest: ok=%v err=%v", ok, err)
	}
	if string(v.Content) != "export X=1\n" || v.CapturedOn != "np10" || v.Generation != 1 || v.CapturedAt.IsZero() {
		t.Fatalf("version after reload: %+v", v)
	}
	if re.Header.Generation != 1 {
		t.Fatalf("reloaded generation %d, want 1", re.Header.Generation)
	}
}

func TestStoreVersionsDuplicatesAndCascade(t *testing.T) {
	dir := t.TempDir()
	st, _ := newTestStore(t, dir)
	defer st.Close()
	e, err := st.AddEntry("$XDG_CONFIG_HOME/git/config", 0o600, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddEntry("$XDG_CONFIG_HOME/git/config", 0o600, ""); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate unbound entry: got %v, want ErrExists", err)
	}
	if _, err := st.AddEntry("$XDG_CONFIG_HOME/git/config", 0o600, "np10"); err != nil {
		t.Fatalf("host-bound entry for the same target must be allowed: %v", err)
	}
	if _, ok, err := st.Latest(e.ID); err != nil || ok {
		t.Fatalf("latest on empty entry: ok=%v err=%v", ok, err)
	}
	v1, err := st.AddVersion(e.ID, []byte("one"), "a")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := st.AddVersion(e.ID, []byte("two"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if v1.Generation != 1 || v2.Generation != 2 || v1.SHA256 == v2.SHA256 {
		t.Fatalf("versions: %+v %+v", v1, v2)
	}
	latest, _, _ := st.Latest(e.ID)
	if !bytes.Equal(latest.Content, []byte("two")) {
		t.Fatalf("latest content %q", latest.Content)
	}
	if err := st.SetMode(e.ID, 0o644); err != nil {
		t.Fatal(err)
	}
	entries, _ := st.Entries()
	if entries[0].Mode != 0o644 {
		t.Fatalf("mode after SetMode %o", entries[0].Mode)
	}
	if err := st.RemoveEntry(e.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := st.VersionCount(e.ID); n != 0 {
		t.Fatalf("versions survived entry removal: %d", n)
	}
	if err := st.RemoveEntry(e.ID); err == nil {
		t.Fatal("removing a missing entry must fail")
	}
}

func TestStoreSaveDetectsConcurrentWriter(t *testing.T) {
	dir := t.TempDir()
	st, key := newTestStore(t, dir)
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	st.Close()
	path := filepath.Join(dir, "macfit.store")
	first, err := Load(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Load(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := first.AddEntry("~/.a", 0o644, ""); err != nil {
		t.Fatal(err)
	}
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := second.AddEntry("~/.b", 0o644, ""); err != nil {
		t.Fatal(err)
	}
	if err := second.Save(); !errors.Is(err, ErrConflict) {
		t.Fatalf("second writer: got %v, want ErrConflict", err)
	}
	check, err := Load(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	entries, _ := check.Entries()
	if len(entries) != 1 || entries[0].Target != "~/.a" {
		t.Fatalf("store on disk holds %+v, want only ~/.a", entries)
	}
}

func TestLoadRejectsWrongKeyAndForeignFile(t *testing.T) {
	dir := t.TempDir()
	st, _ := newTestStore(t, dir)
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	st.Close()
	path := filepath.Join(dir, "macfit.store")
	if _, err := Load(path, mustKey(t)); !errors.Is(err, ErrAuth) {
		t.Fatalf("wrong key: got %v", err)
	}
	other := filepath.Join(dir, "plain.txt")
	os.WriteFile(other, []byte("not a store"), 0o600)
	if _, err := Load(other, mustKey(t)); !errors.Is(err, ErrFormat) {
		t.Fatalf("foreign file: got %v", err)
	}
}
