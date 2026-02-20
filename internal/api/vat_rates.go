package api

import (
	"encoding/json"
	"net/http"
)

type VATRate struct {
	Rate      float64 `json:"rate"`
	Name      string  `json:"name,omitempty"`
	IsDefault bool    `json:"is_default,omitempty"`
}

var defaultVATRates = []VATRate{
	{Rate: 0},
	{Rate: 12},
	{Rate: 21, IsDefault: true},
}

func (s *Server) getVATRates(w http.ResponseWriter, r *http.Request) {
	raw, err := s.settings.Get("vat_rates")
	if err != nil || raw == "" {
		writeJSON(w, http.StatusOK, defaultVATRates)
		return
	}

	var rates []VATRate
	if err := json.Unmarshal([]byte(raw), &rates); err != nil {
		writeJSON(w, http.StatusOK, defaultVATRates)
		return
	}

	writeJSON(w, http.StatusOK, rates)
}

func (s *Server) updateVATRates(w http.ResponseWriter, r *http.Request) {
	var rates []VATRate
	if err := readJSON(r, &rates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(rates) == 0 {
		writeError(w, http.StatusBadRequest, "at least one VAT rate is required")
		return
	}

	data, err := json.Marshal(rates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.settings.Set("vat_rates", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, rates)
}
