package keychain

import (
	"errors"
	"os"
	"testing"
)

func TestOSStoreRoundTrip(t *testing.T) {
	if os.Getenv("TIDYBILL_KEYCHAIN_TEST") != "1" {
		t.Skip("set TIDYBILL_KEYCHAIN_TEST=1 to run")
	}
	store := &osStore{}
	account := "test.keychain.roundtrip"

	if err := store.Set(account, "hello"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(account)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "hello" {
		t.Fatalf("Get returned %q, want %q", got, "hello")
	}

	if err := store.Delete(account); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Get(account)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: got err %v, want ErrNotFound", err)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()

	store, err := newFileStore(dir)
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}

	account := "test.file.roundtrip"

	// Set and Get
	if err := store.Set(account, "world"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get(account)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "world" {
		t.Fatalf("Get returned %q, want %q", got, "world")
	}

	// Values persist across a fresh fileStore at the same path
	store2, err := newFileStore(dir)
	if err != nil {
		t.Fatalf("newFileStore (2nd): %v", err)
	}
	got2, err := store2.Get(account)
	if err != nil {
		t.Fatalf("Get on 2nd store: %v", err)
	}
	if got2 != "world" {
		t.Fatalf("2nd store Get returned %q, want %q", got2, "world")
	}

	// Delete then Get returns ErrNotFound
	if err := store.Delete(account); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.Get(account)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: got err %v, want ErrNotFound", err)
	}
}
