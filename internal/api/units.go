package api

import (
	"encoding/json"
	"net/http"
)

type Unit struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
}

var defaultUnits = []Unit{
	{Name: "ks", IsDefault: true},
	{Name: "hod"},
	{Name: "den"},
	{Name: "m\u00B2"},
}

func (s *Server) getUnits(w http.ResponseWriter, r *http.Request) {
	raw, err := s.settings.Get("units")
	if err != nil || raw == "" {
		writeJSON(w, http.StatusOK, defaultUnits)
		return
	}

	var units []Unit
	if err := json.Unmarshal([]byte(raw), &units); err != nil {
		writeJSON(w, http.StatusOK, defaultUnits)
		return
	}

	writeJSON(w, http.StatusOK, units)
}

func (s *Server) updateUnits(w http.ResponseWriter, r *http.Request) {
	var units []Unit
	if err := readJSON(r, &units); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(units) == 0 {
		writeError(w, http.StatusBadRequest, "at least one unit is required")
		return
	}

	data, err := json.Marshal(units)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.settings.Set("units", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, units)
}
