package api

import (
	"encoding/json"
	"net/http"
)

type Currency struct {
	Code string `json:"code"`
}

var defaultCurrencies = []Currency{
	{Code: "CZK"},
	{Code: "EUR"},
	{Code: "USD"},
	{Code: "GBP"},
	{Code: "PLN"},
	{Code: "CHF"},
}

func (s *Server) getCurrencies(w http.ResponseWriter, r *http.Request) {
	raw, err := s.settings.Get("currencies")
	if err != nil || raw == "" {
		writeJSON(w, http.StatusOK, defaultCurrencies)
		return
	}

	var currencies []Currency
	if err := json.Unmarshal([]byte(raw), &currencies); err != nil {
		writeJSON(w, http.StatusOK, defaultCurrencies)
		return
	}

	writeJSON(w, http.StatusOK, currencies)
}

func (s *Server) updateCurrencies(w http.ResponseWriter, r *http.Request) {
	var currencies []Currency
	if err := readJSON(r, &currencies); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(currencies) == 0 {
		writeError(w, http.StatusBadRequest, "at least one currency is required")
		return
	}

	data, err := json.Marshal(currencies)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.settings.Set("currencies", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, currencies)
}
