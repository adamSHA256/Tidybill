package backup

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	original := []byte(`{"invoices":[{"id":"123","number":"VF26-00001"}]}`)
	passphrase := "test-heslo-123"

	encrypted, err := EncryptExport(original, passphrase)
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	// Should start with magic bytes.
	if !IsEncrypted(encrypted) {
		t.Fatal("encrypted data should start with magic bytes")
	}

	// Should be larger than original (header + tag overhead).
	if len(encrypted) <= len(original) {
		t.Fatalf("encrypted (%d bytes) should be larger than original (%d bytes)", len(encrypted), len(original))
	}

	// Header must be exactly 55 bytes.
	if len(encrypted) < headerLen {
		t.Fatalf("encrypted data (%d bytes) shorter than header (%d bytes)", len(encrypted), headerLen)
	}

	decrypted, err := DecryptExport(encrypted, passphrase)
	if err != nil {
		t.Fatalf("DecryptExport: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Fatalf("decrypted data doesn't match original")
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	original := []byte(`{"test": true}`)
	encrypted, err := EncryptExport(original, "correct-passphrase")
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	_, err = DecryptExport(encrypted, "wrong-passphrase")
	if err == nil {
		t.Fatal("expected error with wrong passphrase")
	}
	if err.Error() != "decryption failed: wrong passphrase or corrupted file" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	original := []byte(`{"test": true}`)
	encrypted, err := EncryptExport(original, "passphrase")
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	// Corrupt a byte in the ciphertext area.
	encrypted[headerLen+5] ^= 0xFF

	_, err = DecryptExport(encrypted, "passphrase")
	if err == nil {
		t.Fatal("expected error with corrupted data")
	}
}

func TestEncryptEmptyPassphrase(t *testing.T) {
	_, err := EncryptExport([]byte(`{}`), "")
	if err == nil {
		t.Fatal("expected error with empty passphrase")
	}
}

func TestEncryptShortPassphrase(t *testing.T) {
	_, err := EncryptExport([]byte(`{}`), "short")
	if err == nil {
		t.Fatal("expected error with short passphrase")
	}
	if err.Error() != "passphrase must be at least 8 characters" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptEmptyPassphrase(t *testing.T) {
	_, err := DecryptExport([]byte("TBILL\x01"+string(make([]byte, 100))), "")
	if err == nil {
		t.Fatal("expected error with empty passphrase")
	}
}

func TestDecryptShortPassphrase(t *testing.T) {
	_, err := DecryptExport([]byte("TBILL\x01"+string(make([]byte, 100))), "short")
	if err == nil {
		t.Fatal("expected error with short passphrase")
	}
	if err.Error() != "passphrase must be at least 8 characters" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := DecryptExport([]byte("TBILL\x01short"), "passphrase")
	if err == nil {
		t.Fatal("expected error with too-short data")
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"encrypted", []byte("TBILL\x01restofdata..."), true},
		{"plain json", []byte(`{"invoices":[]}`), false},
		{"empty", []byte{}, false},
		{"too short", []byte("TBI"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEncrypted(tt.data); got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDifferentEncryptionsProduceDifferentOutput(t *testing.T) {
	data := []byte(`{"same": "data"}`)
	passphrase := "same-passphrase"

	enc1, _ := EncryptExport(data, passphrase)
	enc2, _ := EncryptExport(data, passphrase)

	if bytes.Equal(enc1, enc2) {
		t.Fatal("two encryptions of the same data should produce different output (random salt + nonce)")
	}

	// But both should decrypt to the same thing.
	dec1, _ := DecryptExport(enc1, passphrase)
	dec2, _ := DecryptExport(enc2, passphrase)

	if !bytes.Equal(dec1, dec2) {
		t.Fatal("both should decrypt to the same original data")
	}
}

func TestHeaderLength(t *testing.T) {
	// Verify the header is exactly 55 bytes: magic(6) + salt(16) + time(4) + memory(4) + threads(1) + nonce(24).
	expected := 6 + 16 + 4 + 4 + 1 + 24
	if headerLen != expected {
		t.Fatalf("headerLen = %d, want %d", headerLen, expected)
	}
	if headerLen != 55 {
		t.Fatalf("headerLen = %d, want 55", headerLen)
	}
}

func TestGenerateRecoveryMnemonic(t *testing.T) {
	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}

	// Should be 12 words.
	words := bytes.Fields([]byte(mnemonic))
	if len(words) != 12 {
		t.Fatalf("expected 12 words, got %d: %s", len(words), mnemonic)
	}

	// Should be valid.
	if !ValidateMnemonic(mnemonic) {
		t.Fatalf("generated mnemonic should be valid: %s", mnemonic)
	}
}

func TestValidateMnemonic(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		want     bool
	}{
		{"invalid single word", "hello", false},
		{"invalid gibberish", "aaa bbb ccc ddd eee fff ggg hhh iii jjj kkk lll", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateMnemonic(tt.mnemonic); got != tt.want {
				t.Errorf("ValidateMnemonic() = %v, want %v", got, tt.want)
			}
		})
	}

	// Generate a valid mnemonic and confirm it validates.
	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}
	if !ValidateMnemonic(mnemonic) {
		t.Fatalf("freshly generated mnemonic should be valid: %s", mnemonic)
	}
}

func TestMnemonicCanBeUsedAsPassphrase(t *testing.T) {
	original := []byte(`{"test": "mnemonic-as-passphrase"}`)

	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}

	// Encrypt with the mnemonic as passphrase.
	encrypted, err := EncryptExport(original, mnemonic)
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	// Decrypt with the same mnemonic.
	decrypted, err := DecryptExport(encrypted, mnemonic)
	if err != nil {
		t.Fatalf("DecryptExport: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Fatal("round-trip with mnemonic as passphrase failed")
	}
}

func TestHighMemoryDetection(t *testing.T) {
	data := []byte(`{"test": true}`)
	encrypted, err := EncryptExport(data, "testpassword!")
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	// Modify the memory param in the header to something huge (512 MiB = 524288 KiB).
	// Memory is at offset: magic(6) + salt(16) + time(4) = 26, 4 bytes, little-endian.
	binary.LittleEndian.PutUint32(encrypted[26:30], 524288) // 512 MiB

	// Attempt to decrypt -- should return HighMemoryError.
	_, err = DecryptExport(encrypted, "testpassword!")
	if err == nil {
		t.Fatal("expected HighMemoryError, got nil")
	}

	var highMemErr *HighMemoryError
	if !errors.As(err, &highMemErr) {
		t.Fatalf("expected *HighMemoryError, got %T: %v", err, err)
	}

	if highMemErr.RequestedMiB != 512 {
		t.Fatalf("expected RequestedMiB=512, got %d", highMemErr.RequestedMiB)
	}

	// Should also match the sentinel via errors.Is.
	if !errors.Is(err, ErrHighMemory) {
		t.Fatal("expected error to match ErrHighMemory sentinel")
	}
}
