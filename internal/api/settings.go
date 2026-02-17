package api

import (
	"net/http"
	"os"
)

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	lang, _ := s.settings.Get("language")
	if lang == "" {
		lang = "cs"
	}

	dirLogos, _ := s.settings.Get("dir.logos")
	dirPdfs, _ := s.settings.Get("dir.pdfs")
	dirPreviews, _ := s.settings.Get("dir.previews")
	defaultCurrency, _ := s.settings.Get("default.currency")
	if defaultCurrency == "" {
		defaultCurrency = "CZK"
	}
	defaultDueDays, _ := s.settings.Get("default.due_days")
	if defaultDueDays == "" {
		defaultDueDays = "14"
	}
	defaultVatRate, _ := s.settings.Get("default.vat_rate")
	if defaultVatRate == "" {
		defaultVatRate = "21"
	}
	dashboardWidgets, _ := s.settings.Get("dashboard.widgets")
	customCurrencies, _ := s.settings.Get("custom.currencies")
	customCountries, _ := s.settings.Get("custom.countries")
	invoiceDefaultSort, _ := s.settings.Get("invoice.default_sort") // TODO: also expose in CLI settings menu
	uiScale, _ := s.settings.Get("ui.scale")

	writeJSON(w, http.StatusOK, map[string]string{
		"language":             lang,
		"dir_logos":            dirLogos,
		"dir_pdfs":             dirPdfs,
		"dir_previews":         dirPreviews,
		"default_currency":     defaultCurrency,
		"default_due_days":     defaultDueDays,
		"default_vat_rate":     defaultVatRate,
		"dashboard_widgets":    dashboardWidgets,
		"custom_currencies":    customCurrencies,
		"custom_countries":     customCountries,
		"invoice_default_sort": invoiceDefaultSort,
		"ui_scale":             uiScale,
		"default_pdf_dir":      s.cfg.PDFDir,
	})
}

type UpdateSettingsRequest struct {
	Language         string  `json:"language"`
	DirLogos         *string `json:"dir_logos"`
	DirPdfs          *string `json:"dir_pdfs"`
	DirPreviews      *string `json:"dir_previews"`
	DefaultCurrency  *string `json:"default_currency"`
	DefaultDueDays   *string `json:"default_due_days"`
	DefaultVatRate     *string `json:"default_vat_rate"`
	DashboardWidgets   *string `json:"dashboard_widgets"`
	CustomCurrencies   *string `json:"custom_currencies"`
	CustomCountries    *string `json:"custom_countries"`
	InvoiceDefaultSort *string `json:"invoice_default_sort"` // TODO: also expose in CLI settings menu
	UIScale            *string `json:"ui_scale"`
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

	// Simple key-value settings
	simpleSettings := map[string]*string{
		"default.currency":    req.DefaultCurrency,
		"default.due_days":    req.DefaultDueDays,
		"default.vat_rate":    req.DefaultVatRate,
		"dashboard.widgets":   req.DashboardWidgets,
		"custom.currencies":     req.CustomCurrencies,
		"custom.countries":      req.CustomCountries,
		"invoice.default_sort":  req.InvoiceDefaultSort,
		"ui.scale":              req.UIScale,
	}
	for key, val := range simpleSettings {
		if val == nil {
			continue
		}
		if err := s.settings.Set(key, *val); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Directory settings
	dirSettings := map[string]*string{
		"dir.logos":    req.DirLogos,
		"dir.pdfs":     req.DirPdfs,
		"dir.previews": req.DirPreviews,
	}
	dirFields := map[string]*string{
		"dir.logos":    &s.cfg.LogoDir,
		"dir.pdfs":     &s.cfg.PDFDir,
		"dir.previews": &s.cfg.PreviewDir,
	}

	for key, val := range dirSettings {
		if val == nil {
			continue
		}
		if *val != "" {
			if err := os.MkdirAll(*val, 0755); err != nil {
				writeError(w, http.StatusBadRequest, "cannot create directory: "+err.Error())
				return
			}
		}
		if err := s.settings.Set(key, *val); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Update in-memory config
		if *val != "" {
			*dirFields[key] = *val
		}
	}

	s.getSettings(w, r)
}
