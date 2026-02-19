package api

import (
	"encoding/json"
	"net/http"
)

type DueDaysOption struct {
	Days      int  `json:"days"`
	IsDefault bool `json:"is_default,omitempty"`
}

var defaultDueDaysOptions = []DueDaysOption{
	{Days: 7},
	{Days: 14, IsDefault: true},
	{Days: 30},
	{Days: 60},
}

func (s *Server) getDueDaysOptions(w http.ResponseWriter, r *http.Request) {
	raw, err := s.settings.Get("due_days_options")
	if err != nil || raw == "" {
		writeJSON(w, http.StatusOK, defaultDueDaysOptions)
		return
	}

	var options []DueDaysOption
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		writeJSON(w, http.StatusOK, defaultDueDaysOptions)
		return
	}

	writeJSON(w, http.StatusOK, options)
}

func (s *Server) updateDueDaysOptions(w http.ResponseWriter, r *http.Request) {
	var options []DueDaysOption
	if err := readJSON(r, &options); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(options) == 0 {
		writeError(w, http.StatusBadRequest, "at least one due days option is required")
		return
	}

	data, err := json.Marshal(options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.settings.Set("due_days_options", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, options)
}
