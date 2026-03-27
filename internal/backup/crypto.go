package backup

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// File format constants.
var magicBytes = []byte("TBILL\x01")

const (
	magicLen   = 6
	saltLen    = 16
	timeLen    = 4
	memoryLen  = 4
	threadsLen = 1
	nonceLen   = chacha20poly1305.NonceSizeX // 24
	headerLen  = magicLen + saltLen + timeLen + memoryLen + threadsLen + nonceLen // 55
	keyLen     = chacha20poly1305.KeySize // 32
)

// KDFParams holds Argon2id parameters.
type KDFParams struct {
	Time    uint32
	Memory  uint32 // KiB
	Threads uint8
}

// ErrHighMemory is a sentinel error returned when the file header requests
// more memory than is considered safe for the current device.
var ErrHighMemory = errors.New("file requests high memory for decryption")

// HighMemoryError provides details about the memory mismatch.
type HighMemoryError struct {
	RequestedMiB uint32
	ThresholdMiB uint32
}

func (e *HighMemoryError) Error() string {
	return fmt.Sprintf("file requests %d MiB for decryption, which exceeds the safe limit of %d MiB for this device",
		e.RequestedMiB, e.ThresholdMiB)
}

func (e *HighMemoryError) Is(target error) bool {
	return target == ErrHighMemory
}

// defaultParams returns Argon2id parameters appropriate for the current platform.
func defaultParams() KDFParams {
	if runtime.GOOS == "android" {
		return KDFParams{Time: 1, Memory: 16 * 1024, Threads: 2}
	}
	return KDFParams{Time: 1, Memory: 32 * 1024, Threads: 4}
}

// IsEncrypted checks if a byte slice starts with the TidyBill encryption magic bytes.
func IsEncrypted(data []byte) bool {
	if len(data) < magicLen {
		return false
	}
	return string(data[:magicLen]) == string(magicBytes)
}

// EncryptExport encrypts JSON export data with a user-supplied passphrase.
// Returns the encrypted binary blob in .tidybill format:
// magic(6) + salt(16) + time(4,LE) + memory(4,LE) + threads(1) + nonce(24) + ciphertext+tag
func EncryptExport(jsonData []byte, passphrase string) ([]byte, error) {
	if len(passphrase) < 8 {
		return nil, fmt.Errorf("passphrase must be at least 8 characters")
	}

	params := defaultParams()

	// Generate random salt.
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// Derive key with Argon2id.
	key := argon2.IDKey([]byte(passphrase), salt, params.Time, params.Memory, params.Threads, keyLen)

	// Create XChaCha20-Poly1305 AEAD.
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Generate random nonce.
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt (Seal appends ciphertext + 16-byte Poly1305 tag).
	ciphertext := aead.Seal(nil, nonce, jsonData, nil)

	// Build output: header + ciphertext.
	out := make([]byte, 0, headerLen+len(ciphertext))
	out = append(out, magicBytes...)
	out = append(out, salt...)
	out = binary.LittleEndian.AppendUint32(out, params.Time)
	out = binary.LittleEndian.AppendUint32(out, params.Memory)
	out = append(out, params.Threads)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	// Best-effort memory wipe of key.
	for i := range key {
		key[i] = 0
	}

	return out, nil
}

// DecryptExport decrypts an encrypted .tidybill file using a user-supplied passphrase.
// Returns the decrypted JSON data.
// Returns ErrHighMemory (as *HighMemoryError) if the file's memory parameter exceeds
// a safe threshold for the current device.
func DecryptExport(encData []byte, passphrase string) ([]byte, error) {
	if len(passphrase) < 8 {
		return nil, fmt.Errorf("passphrase must be at least 8 characters")
	}

	// Validate minimum size: header + at least 1 byte ciphertext + 16-byte tag.
	if len(encData) < headerLen+chacha20poly1305.Overhead {
		return nil, errors.New("file is too short or corrupted")
	}

	// Verify magic bytes.
	if string(encData[:magicLen]) != string(magicBytes) {
		return nil, errors.New("not a valid encrypted TidyBill file")
	}

	// Parse header.
	offset := magicLen
	salt := encData[offset : offset+saltLen]
	offset += saltLen

	timeParam := binary.LittleEndian.Uint32(encData[offset : offset+timeLen])
	offset += timeLen

	memoryParam := binary.LittleEndian.Uint32(encData[offset : offset+memoryLen])
	offset += memoryLen

	threads := encData[offset]
	offset += threadsLen

	nonce := encData[offset : offset+nonceLen]
	offset += nonceLen

	ciphertext := encData[offset:]

	// Validate Argon2id params (sanity checks to prevent abuse).
	if timeParam == 0 || timeParam > 10 {
		return nil, fmt.Errorf("invalid KDF time parameter: %d", timeParam)
	}
	if memoryParam < 1024 || memoryParam > 1024*1024 { // 1 MiB to 1 GiB
		return nil, fmt.Errorf("invalid KDF memory parameter: %d KiB", memoryParam)
	}
	if threads == 0 || threads > 16 {
		return nil, fmt.Errorf("invalid KDF threads parameter: %d", threads)
	}

	// Pre-decryption safety check: warn if file header requests more memory
	// than is safe for the current device.
	safeMemoryKiB := uint32(256 * 1024) // 256 MiB for desktop
	if runtime.GOOS == "android" {
		safeMemoryKiB = 64 * 1024 // 64 MiB for mobile
	}
	if memoryParam > safeMemoryKiB {
		return nil, &HighMemoryError{
			RequestedMiB: memoryParam / 1024,
			ThresholdMiB: safeMemoryKiB / 1024,
		}
	}

	// Derive key using params from the file header.
	key := argon2.IDKey([]byte(passphrase), salt, timeParam, memoryParam, threads, keyLen)

	// Create XChaCha20-Poly1305 AEAD.
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Decrypt + authenticate.
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Poly1305 authentication failed -- wrong passphrase or corrupted file.
		return nil, errors.New("decryption failed: wrong passphrase or corrupted file")
	}

	// Best-effort memory wipe of key.
	for i := range key {
		key[i] = 0
	}

	return plaintext, nil
}

// GenerateRecoveryMnemonic generates a 12-word BIP-39 mnemonic (128 bits of entropy).
func GenerateRecoveryMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", fmt.Errorf("generate entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("generate mnemonic: %w", err)
	}
	return mnemonic, nil
}

// ValidateMnemonic checks if a mnemonic is valid (correct word count, valid words,
// valid checksum). Returns false if any word is misspelled or the checksum fails.
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}
