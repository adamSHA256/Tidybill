package keychain

import (
	"strings"
	"testing"
)

// validPhrase is a deterministic 12-word phrase with a correct BIP-39 checksum,
// used to test normalization without depending on a specific test vector.
const validPhrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestNormalizeRecoveryPhrase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{validPhrase, validPhrase},
		{strings.ToUpper(validPhrase), validPhrase},
		{"  " + validPhrase + "  ", validPhrase},
		{strings.ReplaceAll(validPhrase, " ", "  "), validPhrase}, // double spaces collapse
		{"Abandon Abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon About", validPhrase},
		{"\tabandon\nabandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about\n", validPhrase},
	}
	for _, c := range cases {
		got := NormalizeRecoveryPhrase(c.in)
		if got != c.want {
			t.Errorf("NormalizeRecoveryPhrase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSetMasterKey_AcceptsCaseInsensitive(t *testing.T) {
	store := newMemStore()
	if err := SetMasterKey(store, strings.ToUpper(validPhrase)); err != nil {
		t.Fatalf("expected uppercase phrase to be accepted, got %v", err)
	}
	stored, err := store.Get(AcctMasterRecoveryPhrase)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored != validPhrase {
		t.Errorf("expected stored phrase to be normalized lowercase, got %q", stored)
	}
}

func TestSetMasterKey_AcceptsExtraWhitespace(t *testing.T) {
	store := newMemStore()
	if err := SetMasterKey(store, "  "+strings.ReplaceAll(validPhrase, " ", "  ")+"  "); err != nil {
		t.Fatalf("expected whitespace-padded phrase to be accepted, got %v", err)
	}
}

func TestSetMasterKey_RejectsInvalid(t *testing.T) {
	store := newMemStore()
	if err := SetMasterKey(store, "Business businesS not a phrase at all"); err == nil {
		t.Errorf("expected invalid phrase to be rejected")
	}
}

// memStore is a tiny in-memory Store for tests so we don't touch the real
// OS keychain or the file fallback.
type memStore struct{ m map[string]string }

func newMemStore() *memStore { return &memStore{m: map[string]string{}} }

func (s *memStore) Get(account string) (string, error) {
	v, ok := s.m[account]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
func (s *memStore) Set(account, value string) error { s.m[account] = value; return nil }
func (s *memStore) Delete(account string) error     { delete(s.m, account); return nil }
