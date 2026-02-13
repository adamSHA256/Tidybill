package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamSHA256/tidybill/internal/service"
)

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.templates.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.templates.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.templates.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	var req struct {
		Name      *string `json:"name"`
		ShowLogo  *bool   `json:"show_logo"`
		ShowQR    *bool   `json:"show_qr"`
		ShowNotes *bool   `json:"show_notes"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.ShowLogo != nil {
		existing.ShowLogo = *req.ShowLogo
	}
	if req.ShowQR != nil {
		existing.ShowQR = *req.ShowQR
	}
	if req.ShowNotes != nil {
		existing.ShowNotes = *req.ShowNotes
	}

	if err := s.templates.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) setDefaultTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.templates.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	if err := s.templates.SetDefault(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) generateTemplatePreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.templates.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	opts := &service.TemplateOptions{
		ShowLogo:  t.ShowLogo,
		ShowQR:    t.ShowQR,
		ShowNotes: t.ShowNotes,
		QRType:    "spayd",
	}

	path, err := s.pdf.GeneratePreview(t.TemplateCode, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "preview generation failed: "+err.Error())
		return
	}

	t.PreviewPath = path
	s.templates.Update(t)

	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (s *Server) generateAllPreviews(w http.ResponseWriter, r *http.Request) {
	templates, err := s.templates.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := s.pdf.GenerateAllPreviews(templates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update preview_path in DB for each
	for _, t := range templates {
		if path, ok := results[t.ID]; ok && !strings.HasPrefix(path, "error:") {
			t.PreviewPath = path
			s.templates.Update(t)
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) servePreviewPDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.templates.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	if t.PreviewPath == "" {
		writeError(w, http.StatusNotFound, "no preview generated")
		return
	}

	// Security: validate path is within preview dir
	absPath, err := filepath.Abs(t.PreviewPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid path")
		return
	}
	absPreviewDir, err := filepath.Abs(s.cfg.PreviewDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid config")
		return
	}
	if !strings.HasPrefix(absPath, absPreviewDir+string(os.PathSeparator)) && absPath != absPreviewDir {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Clear stale preview_path
		t.PreviewPath = ""
		s.templates.Update(t)
		writeError(w, http.StatusNotFound, "preview file not found")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, absPath)
}
