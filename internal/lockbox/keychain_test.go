package lockbox

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
)

func TestSecurityKeyStoreBuildsExpectedArguments(t *testing.T) {
	key := bytes.Repeat([]byte{0xab}, KeySize)
	var calls [][]string
	ks := SecurityKeyStore{Exec: func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		if args[0] == "find-generic-password" {
			return []byte(hex.EncodeToString(key) + "\n"), nil
		}
		return nil, nil
	}}
	if err := ks.Put("abc123", key); err != nil {
		t.Fatal(err)
	}
	got, err := ks.Get("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, key) {
		t.Fatal("key read back differs")
	}
	want := [][]string{
		{"security", "add-generic-password", "-a", "abc123", "-s", "macfit", "-w", hex.EncodeToString(key), "-U"},
		{"security", "find-generic-password", "-a", "abc123", "-s", "macfit", "-w"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("security calls\n got %v\nwant %v", calls, want)
	}
}

func TestSecurityKeyStoreMapsMissingItem(t *testing.T) {
	ks := SecurityKeyStore{Exec: func(string, ...string) ([]byte, error) {
		return []byte("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.\n"), errors.New("exit status 44")
	}}
	if _, err := ks.Get("abc"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("got %v, want ErrKeyNotFound", err)
	}
	broken := SecurityKeyStore{Exec: func(string, ...string) ([]byte, error) { return []byte("zz\n"), nil }}
	if _, err := broken.Get("abc"); err == nil {
		t.Fatal("malformed keychain item was accepted")
	}
	failing := SecurityKeyStore{Exec: func(string, ...string) ([]byte, error) { return []byte("locked"), errors.New("exit status 1") }}
	if _, err := failing.Get("abc"); err == nil || errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("other failure reported as %v", err)
	}
}

func TestMemoryKeyStore(t *testing.T) {
	var m MemoryKeyStore
	if _, err := m.Get("x"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("empty store: got %v", err)
	}
	key := bytes.Repeat([]byte{1}, KeySize)
	if err := m.Put("x", key); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get("x")
	if err != nil || !bytes.Equal(got, key) {
		t.Fatalf("get after put: %v %v", got, err)
	}
}
