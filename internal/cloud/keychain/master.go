package keychain

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/tyler-smith/go-bip39"
)

// NormalizeRecoveryPhrase trims, lowercases, and collapses internal whitespace
// so user-typed phrases match the canonical BIP-39 wordlist. The wordlist is
// all-lowercase; without this step "Business" or "  business " would fail
// IsMnemonicValid even though they're the user's intent.
//
// Per BIP-39 the comparison IS case-sensitive (each word is exactly the
// lowercase form), so we don't change semantics — we just meet users where
// they type.
func NormalizeRecoveryPhrase(phrase string) string {
	return strings.ToLower(strings.Join(strings.Fields(phrase), " "))
}

// AcctMasterRecoveryPhrase is the keychain account name for the BIP-39
// master recovery phrase. Single entry per app instance; never rotated
// without explicit user action.
const AcctMasterRecoveryPhrase = "tidybill.master.recovery_phrase"

// ErrNoMasterKey is returned when no master recovery phrase is stored.
var ErrNoMasterKey = errors.New("master recovery phrase not configured")

// GetMasterKey retrieves the stored BIP-39 phrase and derives the 64-byte
// seed used as Argon2id input for master-key backups.
// Returns ErrNoMasterKey if no phrase is stored.
// The phrase is not logged or included in returned errors.
func GetMasterKey(kc Store) (phrase string, seed []byte, err error) {
	phrase, err = kc.Get(AcctMasterRecoveryPhrase)
	if errors.Is(err, ErrNotFound) {
		return "", nil, ErrNoMasterKey
	}
	if err != nil {
		return "", nil, fmt.Errorf("keychain read: %w", err)
	}
	// Derive seed with empty passphrase (the 12 words are the full secret).
	seed = bip39.NewSeed(phrase, "")
	return phrase, seed, nil
}

// SetMasterKey validates the BIP-39 checksum and persists the phrase.
// The phrase is normalized (lowercase + whitespace) before validation, so
// users can type "Business" or "  business  " and still match the all-
// lowercase BIP-39 wordlist. Returns an error if the phrase is invalid or
// the keychain write fails.
func SetMasterKey(kc Store, phrase string) error {
	phrase = NormalizeRecoveryPhrase(phrase)
	if !bip39.IsMnemonicValid(phrase) {
		return errors.New("invalid BIP-39 recovery phrase")
	}
	if err := kc.Set(AcctMasterRecoveryPhrase, phrase); err != nil {
		// Log the underlying error so platform-specific failures (Android
		// fileStore permissions, OS keychain unavailable) are diagnosable
		// from the app log without leaking the phrase itself.
		log.Printf("keychain: SetMasterKey write failed: %v", err)
		return fmt.Errorf("keychain write: %w", err)
	}
	return nil
}

// DeleteMasterKey removes the master recovery phrase from the keychain.
func DeleteMasterKey(kc Store) error {
	if err := kc.Delete(AcctMasterRecoveryPhrase); err != nil {
		return fmt.Errorf("keychain delete: %w", err)
	}
	return nil
}
