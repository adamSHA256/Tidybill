package backup

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/tyler-smith/go-bip39"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	original := []byte(`{"invoices":[{"id":"123","number":"VF26-00001"}]}`)
	passphrase := "test-heslo-123"

	encrypted, err := EncryptExport(original, passphrase)
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

	// Should start with v1 magic bytes.
	if !IsEncrypted(encrypted) {
		t.Fatal("encrypted data should start with magic bytes")
	}
	if string(encrypted[:magicLen]) != string(magicBytesV1) {
		t.Fatal("EncryptExport should write v1 magic")
	}

	// Should be larger than original (header + tag overhead).
	if len(encrypted) <= len(original) {
		t.Fatalf("encrypted (%d bytes) should be larger than original (%d bytes)", len(encrypted), len(original))
	}

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

func TestEncryptDecryptMasterRoundTrip(t *testing.T) {
	original := []byte(`{"invoices":[{"id":"abc","number":"VF26-00002"}]}`)

	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}
	seed := bip39.NewSeed(mnemonic, "")

	encrypted, err := EncryptExportMaster(original, seed)
	if err != nil {
		t.Fatalf("EncryptExportMaster: %v", err)
	}

	// Should start with v2 magic bytes.
	if !IsEncrypted(encrypted) {
		t.Fatal("encrypted data should be recognised as encrypted")
	}
	if string(encrypted[:magicLen]) != string(magicBytesV2) {
		t.Fatal("EncryptExportMaster should write v2 magic")
	}
	if encrypted[magicLen] != byte(EncryptModeMaster) {
		t.Fatalf("expected mode byte %d, got %d", EncryptModeMaster, encrypted[magicLen])
	}

	decrypted, err := DecryptExportMaster(encrypted, seed)
	if err != nil {
		t.Fatalf("DecryptExportMaster: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Fatalf("decrypted data doesn't match original")
	}
}

func TestDetectEncryptMode(t *testing.T) {
	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}
	seed := bip39.NewSeed(mnemonic, "")

	v1, _ := EncryptExport([]byte(`{}`), "passphrase123")
	v2master, _ := EncryptExportMaster([]byte(`{}`), seed)

	mode, err := DetectEncryptMode(v1)
	if err != nil {
		t.Fatalf("DetectEncryptMode v1: %v", err)
	}
	if mode != EncryptModeLegacy {
		t.Fatalf("v1 should detect as legacy, got %d", mode)
	}

	mode, err = DetectEncryptMode(v2master)
	if err != nil {
		t.Fatalf("DetectEncryptMode v2master: %v", err)
	}
	if mode != EncryptModeMaster {
		t.Fatalf("v2 master should detect as master, got %d", mode)
	}

	_, err = DetectEncryptMode([]byte(`{"not":"encrypted"}`))
	if err == nil {
		t.Fatal("expected error for plain JSON")
	}
}

func TestDecryptWrongFunction(t *testing.T) {
	mnemonic, _ := GenerateRecoveryMnemonic()
	seed := bip39.NewSeed(mnemonic, "")
	data := []byte(`{"test": true}`)

	// Encrypt with master, try to decrypt with passphrase.
	masterBlob, _ := EncryptExportMaster(data, seed)
	_, err := DecryptExport(masterBlob, "passphrase123")
	if !errors.Is(err, ErrWrongDecryptFunc) {
		t.Fatalf("expected ErrWrongDecryptFunc, got %v", err)
	}

	// Encrypt with passphrase (v1), try to decrypt with master.
	v1blob, _ := EncryptExport(data, "passphrase123")
	_, err = DecryptExportMaster(v1blob, seed)
	if !errors.Is(err, ErrWrongDecryptFunc) {
		t.Fatalf("expected ErrWrongDecryptFunc for v1 blob, got %v", err)
	}
}

func TestDecryptMasterWrongSeed(t *testing.T) {
	mnemonic1, _ := GenerateRecoveryMnemonic()
	mnemonic2, _ := GenerateRecoveryMnemonic()
	seed1 := bip39.NewSeed(mnemonic1, "")
	seed2 := bip39.NewSeed(mnemonic2, "")

	data := []byte(`{"test": true}`)
	encrypted, _ := EncryptExportMaster(data, seed1)

	_, err := DecryptExportMaster(encrypted, seed2)
	if err == nil {
		t.Fatal("expected error decrypting with wrong seed")
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
		{"v1 encrypted", []byte("TBILL\x01restofdata..."), true},
		{"v2 encrypted", []byte("TBILL\x02\x01restofdata..."), true},
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

	dec1, _ := DecryptExport(enc1, passphrase)
	dec2, _ := DecryptExport(enc2, passphrase)

	if !bytes.Equal(dec1, dec2) {
		t.Fatal("both should decrypt to the same original data")
	}
}

func TestHeaderLength(t *testing.T) {
	// v1 header: magic(6) + salt(16) + time(4) + memory(4) + threads(1) + nonce(24) = 55.
	expected := 6 + 16 + 4 + 4 + 1 + 24
	if headerLen != expected {
		t.Fatalf("headerLen = %d, want %d", headerLen, expected)
	}
	if headerLen != 55 {
		t.Fatalf("headerLen = %d, want 55", headerLen)
	}
	// v2 header is one byte longer (mode byte).
	if headerLenV2 != 56 {
		t.Fatalf("headerLenV2 = %d, want 56", headerLenV2)
	}
}

func TestGenerateRecoveryMnemonic(t *testing.T) {
	mnemonic, err := GenerateRecoveryMnemonic()
	if err != nil {
		t.Fatalf("GenerateRecoveryMnemonic: %v", err)
	}

	words := bytes.Fields([]byte(mnemonic))
	if len(words) != 12 {
		t.Fatalf("expected 12 words, got %d: %s", len(words), mnemonic)
	}

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

	encrypted, err := EncryptExport(original, mnemonic)
	if err != nil {
		t.Fatalf("EncryptExport: %v", err)
	}

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

	// Modify the memory param in the v1 header to something huge (512 MiB = 524288 KiB).
	// Memory is at offset: magic(6) + salt(16) + time(4) = 26, 4 bytes, little-endian.
	binary.LittleEndian.PutUint32(encrypted[26:30], 524288) // 512 MiB

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

	if !errors.Is(err, ErrHighMemory) {
		t.Fatal("expected error to match ErrHighMemory sentinel")
	}
}

func TestHighMemoryDetectionV2(t *testing.T) {
	mnemonic, _ := GenerateRecoveryMnemonic()
	seed := bip39.NewSeed(mnemonic, "")
	data := []byte(`{"test": true}`)

	encrypted, err := EncryptExportMaster(data, seed)
	if err != nil {
		t.Fatalf("EncryptExportMaster: %v", err)
	}

	// v2 memory is at offset: magic(6) + mode(1) + salt(16) + time(4) = 27, 4 bytes.
	binary.LittleEndian.PutUint32(encrypted[27:31], 524288)

	_, err = DecryptExportMaster(encrypted, seed)
	if err == nil {
		t.Fatal("expected HighMemoryError, got nil")
	}
	if !errors.Is(err, ErrHighMemory) {
		t.Fatalf("expected ErrHighMemory, got %v", err)
	}
}
