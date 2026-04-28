package keychain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"

	"github.com/adamSHA256/tidybill/internal/backup"
)

// Account names used across transports. Single source of truth.
const (
	AcctGDriveRefreshToken = "cloud.gdrive.refresh_token"

	// rclone per-backend credentials. <backend> is e.g. "sftp", "webdav".
	// Field names inside the envelope follow the rclone CLI option names.
	AcctRcloneTemplate = "cloud.rclone.%s.%s" // fmt: backend, field

	AcctCachedPassphrase = "cloud.passphrase.remembered"
)

func RcloneAcct(backend, field string) string {
	return fmt.Sprintf(AcctRcloneTemplate, backend, field)
}

// ServiceName is the constant used for every keychain entry TidyBill
// creates. Do NOT change this after release — rotating it orphans
// users' existing credentials.
const ServiceName = "TidyBill"

// ErrNotFound is returned by Get when no entry exists.
var ErrNotFound = errors.New("keychain: entry not found")

// Store abstracts the backing store (real OS keychain or encrypted-file fallback).
type Store interface {
	Get(account string) (string, error)
	Set(account, value string) error
	Delete(account string) error
}

var (
	probeOnce sync.Once
	probeOK   bool
)

// New returns the best available Store for this machine:
// real OS keychain if accessible, otherwise a file-backed fallback in dataDir.
// The keychain probe runs ONCE per process and is memoised — this
// prevents libsecret (and equivalents) from prompting the user for
// their keyring passphrase on every operation.
func New(dataDir string) (Store, error) {
	probeOnce.Do(func() { probeOK = osKeychainAvailable() })
	if probeOK {
		return &osStore{}, nil
	}
	log.Printf("keychain: OS keychain unavailable; using encrypted file fallback at %s", dataDir)
	return newFileStore(dataDir)
}

// --- real OS keychain backend ---

type osStore struct{}

func (s *osStore) Get(account string) (string, error) {
	v, err := keyring.Get(ServiceName, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return v, err
}

func (s *osStore) Set(account, value string) error {
	return keyring.Set(ServiceName, account, value)
}

func (s *osStore) Delete(account string) error {
	err := keyring.Delete(ServiceName, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// osKeychainAvailable probes the keychain by performing a harmless
// Set+Get+Delete on a probe key. If any step fails we use the fallback.
// CALLED ONCE per process from New's sync.Once.
func osKeychainAvailable() bool {
	probe := "_probe_" + hex.EncodeToString(randBytes(4))
	if err := keyring.Set(ServiceName, probe, "ok"); err != nil {
		return false
	}
	_, getErr := keyring.Get(ServiceName, probe)
	_ = keyring.Delete(ServiceName, probe)
	return getErr == nil
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// --- file-backed fallback ---
// Stored as a single TBILL\x01-encrypted JSON map keyed by account.
// Passphrase is derived from a machine identifier (see deriveMachineKey).

type fileStore struct {
	path       string
	machineKey []byte

	mu     sync.Mutex
	cached map[string]string
	loaded bool
}

func newFileStore(dataDir string) (*fileStore, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("keychain: empty dataDir for file fallback")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("keychain fileStore: mkdir %q: %w", dataDir, err)
	}
	// Write probe — fail fast if Android filesDir or any other dataDir
	// returns EACCES / EROFS / ENOSPC, so the keychain init returns a
	// useful error instead of every later Set() reporting it. Keeps the
	// "Chyba úložiště klíčů" branch firing at startup with a precise
	// reason rather than mid-flow during phrase generation.
	probePath := filepath.Join(dataDir, ".keychain-probe")
	if err := os.WriteFile(probePath, []byte("ok"), 0o600); err != nil {
		return nil, fmt.Errorf("keychain fileStore: write probe %q: %w", probePath, err)
	}
	_ = os.Remove(probePath)
	key, err := deriveMachineKey(dataDir)
	if err != nil {
		return nil, fmt.Errorf("keychain fileStore: derive machine key: %w", err)
	}
	return &fileStore{
		path:       filepath.Join(dataDir, "credentials.enc"),
		machineKey: key,
	}, nil
}

func (s *fileStore) loadLocked() error {
	if s.loaded {
		return nil
	}
	s.cached = map[string]string{}
	s.loaded = true

	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	plain, err := backup.DecryptExport(raw, string(s.machineKey))
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, &s.cached)
}

func (s *fileStore) saveLocked() error {
	plain, err := json.Marshal(s.cached)
	if err != nil {
		return fmt.Errorf("fileStore marshal: %w", err)
	}
	enc, err := backup.EncryptExport(plain, string(s.machineKey))
	if err != nil {
		return fmt.Errorf("fileStore encrypt: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return fmt.Errorf("fileStore write %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("fileStore rename %q -> %q: %w", tmp, s.path, err)
	}
	return nil
}

func (s *fileStore) Get(account string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return "", err
	}
	v, ok := s.cached[account]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *fileStore) Set(account, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	s.cached[account] = value
	return s.saveLocked()
}

func (s *fileStore) Delete(account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	if _, ok := s.cached[account]; !ok {
		return nil
	}
	delete(s.cached, account)
	return s.saveLocked()
}

// deriveMachineKey returns a stable 32-byte key bound to this machine.
// It is explicitly weaker than a real OS keychain: any process on the
// box running as the same user can decrypt credentials.enc. It exists
// only to avoid plaintext on disk when libsecret (or equivalent) is
// not installed.
//
// Construction: HMAC-SHA256(machineSalt, hostname || "\x00" || "tidybill-fallback-v1").
// machineSalt is 32 random bytes persisted at dataDir/.machine-id with
// mode 0600; it is created on first call and never rotated.
func deriveMachineKey(dataDir string) ([]byte, error) {
	saltPath := filepath.Join(dataDir, ".machine-id")
	salt, err := os.ReadFile(saltPath)
	if errors.Is(err, os.ErrNotExist) {
		salt = randBytes(32)
		if err := os.WriteFile(saltPath, salt, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if len(salt) < 16 {
		return nil, fmt.Errorf("keychain: .machine-id too short")
	}
	host, _ := os.Hostname()
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(host))
	mac.Write([]byte{0})
	mac.Write([]byte("tidybill-fallback-v1"))
	return mac.Sum(nil), nil
}
