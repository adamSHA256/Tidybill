package api

import (
	"encoding/json"
	"net/http"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type PaymentType struct {
	Name      string `json:"name"`
	Code      string `json:"code,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

var builtinPaymentCodes = []string{"bank_transfer", "cash"}

func defaultPaymentTypes() []PaymentType {
	return []PaymentType{
		{Code: "bank_transfer", Name: i18n.T("payment_type.bank_transfer"), IsDefault: true},
		{Code: "cash", Name: i18n.T("payment_type.cash")},
	}
}

func resolvePaymentTypes(types []PaymentType) []PaymentType {
	for i := range types {
		if types[i].Code != "" {
			types[i].Name = i18n.T("payment_type." + types[i].Code)
		}
	}
	present := make(map[string]bool)
	for _, t := range types {
		if t.Code != "" {
			present[t.Code] = true
		}
	}
	for _, code := range builtinPaymentCodes {
		if !present[code] {
			types = append(types, PaymentType{
				Code: code,
				Name: i18n.T("payment_type." + code),
			})
		}
	}
	return types
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

	writeJSON(w, http.StatusOK, resolvePaymentTypes(types))
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
	return resolvePaymentTypes(types)
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

	// Ensure built-in payment types are not removed
	for _, code := range builtinPaymentCodes {
		found := false
		for _, pt := range types {
			if pt.Code == code {
				found = true
				break
			}
		}
		if !found {
			types = append(types, PaymentType{Code: code, Name: i18n.T("payment_type." + code)})
		}
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
