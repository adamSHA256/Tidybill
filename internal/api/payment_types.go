package api

import (
	"encoding/json"
	"net/http"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type PaymentType struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
}

func defaultPaymentTypes() []PaymentType {
	return []PaymentType{
		{Name: i18n.T("payment_type.bank_transfer"), IsDefault: true},
		{Name: i18n.T("payment_type.cash")},
	}
}

func (s *Server) getPaymentTypes(w http.ResponseWriter, r *http.Request) {
	raw, err := s.settings.Get("payment_types")
	if err != nil || raw == "" {
		writeJSON(w, http.StatusOK, defaultPaymentTypes())
		return
	}

	var types []PaymentType
	if err := json.Unmarshal([]byte(raw), &types); err != nil {
		writeJSON(w, http.StatusOK, defaultPaymentTypes())
		return
	}

	writeJSON(w, http.StatusOK, types)
}

func (s *Server) loadPaymentTypes() []PaymentType {
	raw, err := s.settings.Get("payment_types")
	if err != nil || raw == "" {
		return defaultPaymentTypes()
	}
	var types []PaymentType
	if err := json.Unmarshal([]byte(raw), &types); err != nil {
		return defaultPaymentTypes()
	}
	return types
}

func (s *Server) updatePaymentTypes(w http.ResponseWriter, r *http.Request) {
	var types []PaymentType
	if err := readJSON(r, &types); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(types) == 0 {
		writeError(w, http.StatusBadRequest, "at least one payment type is required")
		return
	}

	data, err := json.Marshal(types)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.settings.Set("payment_types", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types)
}
