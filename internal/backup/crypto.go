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
//
// v1 layout: magic(6) + salt(16) + time(4,LE) + mem(4,LE) + threads(1) + nonce(24) + ciphertext+tag
// v2 layout: magic(6) + mode(1) + salt(16) + time(4,LE) + mem(4,LE) + threads(1) + nonce(24) + ciphertext+tag
//
// v1 magic is "TBILL\x01"; v2 magic is "TBILL\x02".
// v1 is read-only (legacy). New exports always use v2 mode 1 (master-key).
var (
	magicBytesV1 = []byte("TBILL\x01")
	magicBytesV2 = []byte("TBILL\x02")
)

const (
	magicLen   = 6
	saltLen    = 16
	timeLen    = 4
	memoryLen  = 4
	threadsLen = 1
	nonceLen   = chacha20poly1305.NonceSizeX // 24
	// v1 header: no mode byte.
	headerLen = magicLen + saltLen + timeLen + memoryLen + threadsLen + nonceLen // 55
	// v2 header: one extra mode byte after magic.
	headerLenV2 = headerLen + 1 // 56
	keyLen      = chacha20poly1305.KeySize // 32
)

// EncryptMode describes how a v2 blob was encrypted.
type EncryptMode byte

const (
	EncryptModeLegacy EncryptMode = 0 // Argon2id from user passphrase
	EncryptModeMaster EncryptMode = 1 // Argon2id from BIP-39 seed bytes
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

// ErrWrongDecryptFunc is returned when the caller uses the wrong decrypt
// function for the file's encryption mode (e.g. calling DecryptExport on a
// v2 master-key blob, or DecryptExportMaster on a passphrase blob).
var ErrWrongDecryptFunc = errors.New("wrong decrypt function for this encryption mode")

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

// IsEncrypted checks if a byte slice starts with a valid TidyBill encryption magic (v1 or v2).
func IsEncrypted(data []byte) bool {
	if len(data) < magicLen {
		return false
	}
	return string(data[:magicLen]) == string(magicBytesV1) ||
		string(data[:magicLen]) == string(magicBytesV2)
}

// DetectEncryptMode returns the encryption mode of an encrypted blob without
// decrypting it. Returns an error if the data is not a valid TidyBill file.
func DetectEncryptMode(data []byte) (EncryptMode, error) {
	if len(data) < magicLen {
		return 0, errors.New("not a valid encrypted TidyBill file")
	}
	switch string(data[:magicLen]) {
	case string(magicBytesV1):
		return EncryptModeLegacy, nil
	case string(magicBytesV2):
		if len(data) < magicLen+1 {
			return 0, errors.New("file is too short or corrupted")
		}
		return EncryptMode(data[magicLen]), nil
	default:
		return 0, errors.New("not a valid encrypted TidyBill file")
	}
}

// EncryptExport encrypts JSON export data with a user-supplied passphrase.
// Writes v1 format for backwards compatibility with the keychain file-store fallback.
// Returns the encrypted binary blob in .tidybill format:
// magic(6) + salt(16) + time(4,LE) + memory(4,LE) + threads(1) + nonce(24) + ciphertext+tag
func EncryptExport(jsonData []byte, passphrase string) ([]byte, error) {
	if len(passphrase) < 8 {
		return nil, fmt.Errorf("passphrase must be at least 8 characters")
	}
	return encryptWithKey(jsonData, []byte(passphrase), magicBytesV1, nil)
}

// EncryptExportMaster encrypts JSON export data using 64 bytes of BIP-39 seed
// material as the Argon2id input. Writes v2 format with mode byte = 1.
// seed must be at least 32 bytes (bip39.NewSeed returns 64 bytes).
func EncryptExportMaster(jsonData []byte, seed []byte) ([]byte, error) {
	if len(seed) < 32 {
		return nil, fmt.Errorf("seed must be at least 32 bytes")
	}
	modeByte := []byte{byte(EncryptModeMaster)}
	return encryptWithKey(jsonData, seed, magicBytesV2, modeByte)
}

// encryptWithKey is the shared encrypt implementation.
// extraHeader bytes (e.g. mode byte for v2) are inserted between magic and salt.
func encryptWithKey(jsonData, key, magic, extraHeader []byte) ([]byte, error) {
	params := defaultParams()

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	derivedKey := argon2.IDKey(key, salt, params.Time, params.Memory, params.Threads, keyLen)

	aead, err := chacha20poly1305.NewX(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, jsonData, nil)

	capacity := magicLen + len(extraHeader) + saltLen + timeLen + memoryLen + threadsLen + nonceLen + len(ciphertext)
	out := make([]byte, 0, capacity)
	out = append(out, magic...)
	out = append(out, extraHeader...)
	out = append(out, salt...)
	out = binary.LittleEndian.AppendUint32(out, params.Time)
	out = binary.LittleEndian.AppendUint32(out, params.Memory)
	out = append(out, params.Threads)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	for i := range derivedKey {
		derivedKey[i] = 0
	}

	return out, nil
}

// DecryptExport decrypts a v1 or v2-mode-0 blob using a user-supplied passphrase.
// Returns ErrWrongDecryptFunc if the blob is v2 mode 1 (master-key); call
// DecryptExportMaster instead.
func DecryptExport(encData []byte, passphrase string) ([]byte, error) {
	if len(passphrase) < 8 {
		return nil, fmt.Errorf("passphrase must be at least 8 characters")
	}

	minLen := headerLen + chacha20poly1305.Overhead
	if len(encData) < minLen {
		return nil, errors.New("file is too short or corrupted")
	}

	switch string(encData[:magicLen]) {
	case string(magicBytesV1):
		// v1: no mode byte, passphrase-derived key.
		return decryptPayload(encData[magicLen:], []byte(passphrase))

	case string(magicBytesV2):
		if len(encData) < headerLenV2+chacha20poly1305.Overhead {
			return nil, errors.New("file is too short or corrupted")
		}
		mode := EncryptMode(encData[magicLen])
		if mode == EncryptModeMaster {
			return nil, ErrWrongDecryptFunc
		}
		// v2 mode 0: mode byte present, passphrase-derived key.
		return decryptPayload(encData[magicLen+1:], []byte(passphrase))

	default:
		return nil, errors.New("not a valid encrypted TidyBill file")
	}
}

// DecryptExportMaster decrypts a v2-mode-1 blob using BIP-39 seed bytes.
// Returns ErrWrongDecryptFunc if the blob is v1 or v2 mode 0.
func DecryptExportMaster(encData []byte, seed []byte) ([]byte, error) {
	if len(seed) < 32 {
		return nil, fmt.Errorf("seed must be at least 32 bytes")
	}

	if len(encData) < headerLenV2+chacha20poly1305.Overhead {
		return nil, errors.New("file is too short or corrupted")
	}

	if string(encData[:magicLen]) != string(magicBytesV2) {
		return nil, ErrWrongDecryptFunc
	}
	mode := EncryptMode(encData[magicLen])
	if mode != EncryptModeMaster {
		return nil, ErrWrongDecryptFunc
	}

	return decryptPayload(encData[magicLen+1:], seed)
}

// decryptPayload parses salt+kdfparams+nonce+ciphertext and decrypts.
// payload starts right after the magic (and mode byte, if present).
func decryptPayload(payload, key []byte) ([]byte, error) {
	minPayload := saltLen + timeLen + memoryLen + threadsLen + nonceLen + chacha20poly1305.Overhead
	if len(payload) < minPayload {
		return nil, errors.New("file is too short or corrupted")
	}

	offset := 0
	salt := payload[offset : offset+saltLen]
	offset += saltLen

	timeParam := binary.LittleEndian.Uint32(payload[offset : offset+timeLen])
	offset += timeLen

	memoryParam := binary.LittleEndian.Uint32(payload[offset : offset+memoryLen])
	offset += memoryLen

	threads := payload[offset]
	offset += threadsLen

	nonce := payload[offset : offset+nonceLen]
	offset += nonceLen

	ciphertext := payload[offset:]

	if timeParam == 0 || timeParam > 10 {
		return nil, fmt.Errorf("invalid KDF time parameter: %d", timeParam)
	}
	if memoryParam < 1024 || memoryParam > 1024*1024 {
		return nil, fmt.Errorf("invalid KDF memory parameter: %d KiB", memoryParam)
	}
	if threads == 0 || threads > 16 {
		return nil, fmt.Errorf("invalid KDF threads parameter: %d", threads)
	}

	safeMemoryKiB := uint32(256 * 1024)
	if runtime.GOOS == "android" {
		safeMemoryKiB = 64 * 1024
	}
	if memoryParam > safeMemoryKiB {
		return nil, &HighMemoryError{
			RequestedMiB: memoryParam / 1024,
			ThresholdMiB: safeMemoryKiB / 1024,
		}
	}

	derivedKey := argon2.IDKey(key, salt, timeParam, memoryParam, threads, keyLen)

	aead, err := chacha20poly1305.NewX(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong passphrase or corrupted file")
	}

	for i := range derivedKey {
		derivedKey[i] = 0
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
