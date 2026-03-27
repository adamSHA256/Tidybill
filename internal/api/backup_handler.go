package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
)

// exportRequest extends ExportFilters with an optional passphrase for encryption.
type exportRequest struct {
	backup.ExportFilters
	Passphrase string `json:"passphrase,omitempty"`
}

// POST /api/backup/export
// Body: optional JSON with ExportFilters fields + optional "passphrase".
// If passphrase is provided, the export is encrypted.
// Returns: ExportFile JSON (or encrypted binary blob).
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	var passphrase string

	if r.ContentLength > 0 {
		var req exportRequest
		if err := readJSON(r, &req); err == nil {
			passphrase = req.Passphrase
			f := req.ExportFilters
			filters = &f
		}
	}

	if passphrase != "" {
		if len(passphrase) < 8 {
			writeError(w, http.StatusBadRequest, "passphrase must be at least 8 characters")
			return
		}
		data, err := s.backupExport.ExportEncryptedJSON(filters, passphrase)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=tidybill-backup.tidybill")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(data); err != nil {
			log.Printf("backup export: failed to write response: %v", err)
		}
		return
	}

	data, err := s.backupExport.ExportJSON(filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=tidybill-backup.tidybill")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("backup export: failed to write response: %v", err)
	}
}

// POST /api/backup/import
// Body: multipart form with "file" field + "mode" field + optional "passphrase" field.
// If the file is encrypted and no passphrase is provided, returns 400.
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

	// Read file data to check for encryption.
	fileData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file: "+err.Error())
		return
	}

	passphrase := r.FormValue("passphrase")

	// If file is encrypted and no passphrase provided, return 400.
	if backup.IsEncrypted(fileData) && passphrase == "" {
		writeError(w, http.StatusBadRequest, "file is encrypted, passphrase required")
		return
	}
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

	// Read file data to check for encryption.
	fileData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file: "+err.Error())
		return
	}

	passphrase := r.FormValue("passphrase")

	// If file is encrypted and no passphrase provided, return 400.
	if backup.IsEncrypted(fileData) && passphrase == "" {
		writeError(w, http.StatusBadRequest, "file is encrypted, passphrase required")
		return
	}

	previewMode := r.FormValue("mode")

	opts := backup.ImportOptions{
		Mode:        backup.ImportModePreview,
		PreviewMode: previewMode,
		Passphrase:  passphrase,
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
// Used by mobile clients that cannot download blobs directly.
// Body: optional JSON with ExportFilters fields + optional "passphrase".
func (s *Server) handleExportToFile(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	var passphrase string

	if r.ContentLength > 0 {
		var req exportRequest
		if err := readJSON(r, &req); err == nil {
			passphrase = req.Passphrase
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
	if passphrase != "" {
		data, err = s.backupExport.ExportEncryptedJSON(filters, passphrase)
	} else {
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
