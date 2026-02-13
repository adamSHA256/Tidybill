package api

import (
	"net/http"
)

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	lang, _ := s.settings.Get("language")
	if lang == "" {
		lang = "cs"
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"language": lang,
	})
}

type UpdateSettingsRequest struct {
	Language string `json:"language"`
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Language != "" {
		valid := map[string]bool{"cs": true, "sk": true, "en": true}
		if !valid[req.Language] {
			writeError(w, http.StatusBadRequest, "invalid language, must be: cs, sk, en")
			return
		}
		if err := s.settings.Set("language", req.Language); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	s.getSettings(w, r)
}
