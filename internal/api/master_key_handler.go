package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
)

// revealToken is a single-use, 30-second token gating the /reveal endpoint.
type revealTokenState struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

// masterKeyRevealToken is stored on the Server for the reveal-gate flow.
// We store it as a pointer on the Server struct (see router.go).

// GET /api/master-key/status
// Response: { configured: bool }
func (s *Server) handleMasterKeyStatus(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
		return
	}
	_, err := s.kc.Get(keychain.AcctMasterRecoveryPhrase)
	configured := err == nil
	writeJSON(w, http.StatusOK, map[string]any{"configured": configured})
}

// POST /api/master-key/generate
// Generates a new BIP-39 phrase, stores it, returns it.
// Response: { phrase: string }
func (s *Server) handleMasterKeyGenerate(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeError(w, http.StatusServiceUnavailable, "keychain unavailable")
		return
	}

	phrase, err := backup.GenerateRecoveryMnemonic()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generate phrase: "+err.Error())
		return
	}

	if err := keychain.SetMasterKey(s.kc, phrase); err != nil {
		writeError(w, http.StatusInternalServerError, "store phrase: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"phrase": phrase})
}

// POST /api/master-key/import
// Body: { phrase: string }
// Validates BIP-39 checksum and stores.
func (s *Server) handleMasterKeyImport(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeError(w, http.StatusServiceUnavailable, "keychain unavailable")
		return
	}

	var body struct {
		Phrase string `json:"phrase"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Phrase == "" {
		writeError(w, http.StatusBadRequest, "phrase required")
		return
	}

	if err := keychain.SetMasterKey(s.kc, body.Phrase); err != nil {
		if errors.Is(err, backup.ErrWrongDecryptFunc) || err.Error() == "keychain write: invalid BIP-39 recovery phrase" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		// SetMasterKey returns a plain error for invalid phrase.
		if err.Error() == "invalid BIP-39 recovery phrase" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/master-key/reveal-token
// Issues a single-use 30-second token required to call /reveal.
// Response: { token: string, expires_at: RFC3339 }
func (s *Server) handleMasterKeyRevealToken(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeError(w, http.StatusServiceUnavailable, "keychain unavailable")
		return
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		writeError(w, http.StatusInternalServerError, "generate token: "+err.Error())
		return
	}
	token := hex.EncodeToString(b)
	expiry := time.Now().Add(30 * time.Second)

	s.revealToken.mu.Lock()
	s.revealToken.token = token
	s.revealToken.expiry = expiry
	s.revealToken.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_at": expiry.UTC().Format(time.RFC3339),
	})
}

// GET /api/master-key/reveal?token=...
// Returns the stored phrase. Requires a valid, unexpired token from /reveal-token.
// Response: { phrase: string }
func (s *Server) handleMasterKeyReveal(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeError(w, http.StatusServiceUnavailable, "keychain unavailable")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token required")
		return
	}

	s.revealToken.mu.Lock()
	storedToken := s.revealToken.token
	expiry := s.revealToken.expiry
	// Consume the token immediately (single-use).
	s.revealToken.token = ""
	s.revealToken.expiry = time.Time{}
	s.revealToken.mu.Unlock()

	if storedToken == "" || token != storedToken || time.Now().After(expiry) {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	phrase, _, err := keychain.GetMasterKey(s.kc)
	if errors.Is(err, keychain.ErrNoMasterKey) {
		writeError(w, http.StatusNotFound, "master key not configured")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "keychain read error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"phrase": phrase})
}

// DELETE /api/master-key
// Wipes the master recovery phrase. Irreversible.
func (s *Server) handleMasterKeyDelete(w http.ResponseWriter, r *http.Request) {
	if s.kc == nil {
		writeError(w, http.StatusServiceUnavailable, "keychain unavailable")
		return
	}

	if err := keychain.DeleteMasterKey(s.kc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
