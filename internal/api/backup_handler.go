package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
)

// exportRequest extends ExportFilters with optional encryption parameters.
// EncryptMaster=true uses the stored master key (v2 format).
// Passphrase provides legacy passphrase-based encryption (v1 format).
type exportRequest struct {
	backup.ExportFilters
	Passphrase    string `json:"passphrase,omitempty"`
	EncryptMaster bool   `json:"encrypt_master,omitempty"`
}

// resolveMasterSeed fetches the master seed from the keychain. Returns nil seed
// and no error when the keychain is unavailable or no key is stored (caller
// decides whether that is an error).
func (s *Server) resolveMasterSeed() ([]byte, error) {
	if s.kc == nil {
		return nil, nil
	}
	_, seed, err := keychain.GetMasterKey(s.kc)
	if errors.Is(err, keychain.ErrNoMasterKey) {
		return nil, nil
	}
	return seed, err
}

// fillMasterSeed populates opts.MasterSeed when the blob is a v2 master-key
// file. Returns an error if the master key is not configured.
func (s *Server) fillMasterSeed(data []byte, opts *backup.ImportOptions) error {
	if !backup.IsEncrypted(data) {
		return nil
	}
	mode, err := backup.DetectEncryptMode(data)
	if err != nil {
		return err
	}
	if mode != backup.EncryptModeMaster {
		return nil
	}
	seed, err := s.resolveMasterSeed()
	if err != nil {
		return fmt.Errorf("keychain error: %w", err)
	}
	if seed == nil {
		return errors.New("master_key_not_configured: this backup requires the master recovery phrase")
	}
	opts.MasterSeed = seed
	return nil
}

// POST /api/backup/export
// Body: optional JSON with ExportFilters + optional "passphrase" or "encrypt_master".
// Returns: ExportFile JSON (or encrypted binary blob).
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	var passphrase string
	var encryptMaster bool

	if r.ContentLength > 0 {
		var req exportRequest
		if err := readJSON(r, &req); err == nil {
			passphrase = req.Passphrase
			encryptMaster = req.EncryptMaster
			f := req.ExportFilters
			filters = &f
		}
	}

	var data []byte
	var err error

	switch {
	case encryptMaster:
		seed, seedErr := s.resolveMasterSeed()
		if seedErr != nil {
			writeError(w, http.StatusInternalServerError, "keychain error: "+seedErr.Error())
			return
		}
		if seed == nil {
			writeError(w, http.StatusBadRequest, "master_key_not_configured")
			return
		}
		data, err = s.backupExport.ExportMasterEncryptedJSON(filters, seed)

	case passphrase != "":
		if len(passphrase) < 8 {
			writeError(w, http.StatusBadRequest, "passphrase must be at least 8 characters")
			return
		}
		data, err = s.backupExport.ExportEncryptedJSON(filters, passphrase)

	default:
		data, err = s.backupExport.ExportJSON(filters)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if encryptMaster || passphrase != "" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=tidybill-backup.tidybill")
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=tidybill-backup.tidybill")
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("backup export: failed to write response: %v", err)
	}
}

// POST /api/backup/import
// Body: multipart form with "file" field + "mode" field + optional "passphrase" field.
// For v2 master-key files the passphrase is not required; the master key is
// fetched automatically from the keychain.
// Returns: ImportReport JSON
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20) // 100 MB limit

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file: "+err.Error())
		return
	}

	passphrase := r.FormValue("passphrase")
	if passphrase != "" && len(passphrase) < 8 {
		writeError(w, http.StatusBadRequest, "passphrase must be at least 8 characters")
		return
	}

	mode := r.FormValue("mode")
	switch mode {
	case "merge", "":
		mode = "smart_merge"
	case "replace":
		mode = "full_replace"
	case "force":
		// already correct
	default:
		writeError(w, http.StatusBadRequest, "invalid import mode, use: merge, replace, or force")
		return
	}

	opts := backup.ImportOptions{
		Mode:       mode,
		Passphrase: passphrase,
	}
	if err := s.fillMasterSeed(fileData, &opts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// If file is encrypted and neither passphrase nor master seed was provided,
	// return a helpful error.
	if backup.IsEncrypted(fileData) && opts.Passphrase == "" && len(opts.MasterSeed) == 0 {
		writeError(w, http.StatusBadRequest, "file is encrypted, passphrase required")
		return
	}

	report, err := s.backupImport.Import(bytes.NewReader(fileData), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// POST /api/backup/import/preview
// Body: multipart form with "file" field + optional "passphrase" field
// Returns: ImportReport JSON (dry run)
func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20) // 100 MB limit

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file: "+err.Error())
		return
	}

	passphrase := r.FormValue("passphrase")
	previewMode := r.FormValue("mode")

	opts := backup.ImportOptions{
		Mode:        backup.ImportModePreview,
		PreviewMode: previewMode,
		Passphrase:  passphrase,
	}
	if err := s.fillMasterSeed(fileData, &opts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if backup.IsEncrypted(fileData) && opts.Passphrase == "" && len(opts.MasterSeed) == 0 {
		writeError(w, http.StatusBadRequest, "file is encrypted, passphrase required")
		return
	}

	report, err := s.backupImport.Import(bytes.NewReader(fileData), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// GET /api/backup/generate-mnemonic
// Returns a 12-word BIP-39 mnemonic that can be used as a strong passphrase.
func (s *Server) handleGenerateMnemonic(w http.ResponseWriter, r *http.Request) {
	mnemonic, err := backup.GenerateRecoveryMnemonic()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"mnemonic": mnemonic})
}

// POST /api/backup/export-file
// Saves export to a file in the exports directory and returns its path.
// Body: optional JSON with ExportFilters + optional "passphrase" or "encrypt_master".
func (s *Server) handleExportToFile(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	var passphrase string
	var encryptMaster bool

	if r.ContentLength > 0 {
		var req exportRequest
		if err := readJSON(r, &req); err == nil {
			passphrase = req.Passphrase
			encryptMaster = req.EncryptMaster
			f := req.ExportFilters
			filters = &f
		}
	}

	if passphrase != "" && len(passphrase) < 8 {
		writeError(w, http.StatusBadRequest, "passphrase must be at least 8 characters")
		return
	}

	var data []byte
	var err error

	switch {
	case encryptMaster:
		seed, seedErr := s.resolveMasterSeed()
		if seedErr != nil {
			writeError(w, http.StatusInternalServerError, "keychain error: "+seedErr.Error())
			return
		}
		if seed == nil {
			writeError(w, http.StatusBadRequest, "master_key_not_configured")
			return
		}
		data, err = s.backupExport.ExportMasterEncryptedJSON(filters, seed)

	case passphrase != "":
		data, err = s.backupExport.ExportEncryptedJSON(filters, passphrase)

	default:
		data, err = s.backupExport.ExportJSON(filters)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("tidybill-backup-%s.tidybill", time.Now().Format("2006-01-02"))
	path := filepath.Join(s.cfg.ExportDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"path": path, "filename": filename})
}
