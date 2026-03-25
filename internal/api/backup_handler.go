package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
)

// POST /api/backup/export
// Body: optional ExportFilters JSON
// Returns: ExportFile JSON
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	// Try to read filters from body (optional)
	if r.ContentLength > 0 {
		var f backup.ExportFilters
		if err := readJSON(r, &f); err == nil {
			filters = &f
		}
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
// Body: multipart form with "file" field + "mode" field
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

	mode := r.FormValue("mode")
	switch mode {
	case "merge", "":
		mode = "smart_merge"
	case "replace":
		mode = "full_replace"
	case "force":
		// already correct
	}

	opts := backup.ImportOptions{
		Mode: mode,
	}

	report, err := s.backupImport.Import(file, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// POST /api/backup/import/preview
// Body: multipart form with "file" field
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

	opts := backup.ImportOptions{
		Mode: backup.ImportModePreview,
	}

	report, err := s.backupImport.Import(file, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// POST /api/backup/export-file
// Saves export to a file in the exports directory and returns its path.
// Used by mobile clients that cannot download blobs directly.
func (s *Server) handleExportToFile(w http.ResponseWriter, r *http.Request) {
	var filters *backup.ExportFilters
	if r.ContentLength > 0 {
		var f backup.ExportFilters
		if err := readJSON(r, &f); err == nil {
			filters = &f
		}
	}

	data, err := s.backupExport.ExportJSON(filters)
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
