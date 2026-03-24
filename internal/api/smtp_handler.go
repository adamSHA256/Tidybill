package api

import (
	"net/http"

	"github.com/adamSHA256/tidybill/internal/email"
	"github.com/adamSHA256/tidybill/internal/model"
)

// GET /api/suppliers/{id}/smtp
func (s *Server) getSmtpConfig(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")
	config, err := s.smtpConfigs.GetBySupplierID(supplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if config == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"configured": false})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

// PUT /api/suppliers/{id}/smtp
// Request body includes all fields + optional "password" field (plaintext)
func (s *Server) upsertSmtpConfig(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")

	var req struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Username    string `json:"username"`
		Password    string `json:"password"` // plaintext, only when setting/changing
		FromName    string `json:"from_name"`
		FromEmail   string `json:"from_email"`
		UseStartTLS bool   `json:"use_starttls"`
		Enabled     bool   `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Host == "" || req.Username == "" || req.FromEmail == "" {
		writeError(w, http.StatusBadRequest, "host, username, and from_email are required")
		return
	}

	config := model.NewSmtpConfig(supplierID)
	config.Host = req.Host
	config.Port = req.Port
	config.Username = req.Username
	config.FromName = req.FromName
	config.FromEmail = req.FromEmail
	config.UseStartTLS = req.UseStartTLS
	config.Enabled = req.Enabled

	// Check if existing config has a password (preserve it if not changing)
	existing, _ := s.smtpConfigs.GetBySupplierID(supplierID)
	if existing != nil {
		config.ID = existing.ID
		config.PasswordEncrypted = existing.PasswordEncrypted
	}

	// Encrypt new password if provided
	if req.Password != "" {
		encrypted, err := email.EncryptPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt password")
			return
		}
		config.PasswordEncrypted = encrypted
	}

	if err := s.smtpConfigs.Upsert(config); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Re-read to get computed fields
	saved, _ := s.smtpConfigs.GetBySupplierID(supplierID)
	if saved != nil {
		writeJSON(w, http.StatusOK, saved)
	} else {
		writeJSON(w, http.StatusOK, config)
	}
}

// DELETE /api/suppliers/{id}/smtp
func (s *Server) deleteSmtpConfig(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")
	if err := s.smtpConfigs.Delete(supplierID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/suppliers/{id}/smtp/test
func (s *Server) testSmtpConnection(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")

	var req struct {
		// Allow testing with provided credentials (before saving)
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		FromName    string `json:"from_name"`
		FromEmail   string `json:"from_email"`
		UseStartTLS bool   `json:"use_starttls"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	config := &model.SmtpConfig{
		SupplierID:  supplierID,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		FromName:    req.FromName,
		FromEmail:   req.FromEmail,
		UseStartTLS: req.UseStartTLS,
	}

	password := req.Password

	// If no password provided in request, try to use saved one
	if password == "" {
		existing, _ := s.smtpConfigs.GetBySupplierID(supplierID)
		if existing != nil && existing.PasswordEncrypted != "" {
			decrypted, err := email.DecryptPassword(existing.PasswordEncrypted)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to decrypt saved password")
				return
			}
			password = decrypted
		}
	}

	if password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	if err := s.emailService.TestConnection(config, password); err != nil {
		writeError(w, http.StatusBadRequest, "connection failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/suppliers/{id}/smtp/copy/{fromId}
func (s *Server) copySmtpConfig(w http.ResponseWriter, r *http.Request) {
	toSupplierID := r.PathValue("id")
	fromSupplierID := r.PathValue("fromId")

	source, err := s.smtpConfigs.GetBySupplierID(fromSupplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if source == nil {
		writeError(w, http.StatusNotFound, "source SMTP config not found")
		return
	}

	// Copy all fields except ID, SupplierID, and password
	config := model.NewSmtpConfig(toSupplierID)
	config.Host = source.Host
	config.Port = source.Port
	config.Username = source.Username
	config.FromName = source.FromName
	config.FromEmail = source.FromEmail
	config.UseStartTLS = source.UseStartTLS
	config.Enabled = source.Enabled
	// Password is NOT copied — user must enter it

	if err := s.smtpConfigs.Upsert(config); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	saved, _ := s.smtpConfigs.GetBySupplierID(toSupplierID)
	if saved != nil {
		writeJSON(w, http.StatusOK, saved)
	} else {
		writeJSON(w, http.StatusOK, config)
	}
}
