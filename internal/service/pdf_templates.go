package service

import (
	"github.com/johnfercher/maroto/v2/pkg/core"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

// TemplateRenderer defines the interface for PDF templates
type TemplateRenderer interface {
	Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions)
	Margins() TemplateMargins
}

// TemplateOptions holds per-render configuration
type TemplateOptions struct {
	ShowLogo  bool
	ShowQR    bool
	ShowNotes bool
	QRType    string // "spayd", "pay_by_square", "epc", "none"
}

// TemplateMargins defines the page margins for a template
type TemplateMargins struct {
	Left  float64
	Top   float64
	Right float64
}

var templateRegistry = map[string]TemplateRenderer{
	"default": &DefaultTemplate{},
	"classic": &ClassicTemplate{},
	"modern":  &ModernTemplate{},
	"minimal": &MinimalTemplate{},
}

func GetTemplateRenderer(code string) TemplateRenderer {
	if r, ok := templateRegistry[code]; ok {
		return r
	}
	return templateRegistry["default"]
}

// invoiceLang extracts the language from invoice data, falling back to the global setting.
func invoiceLang(data *InvoiceData) i18n.Lang {
	if data.Invoice.Language != "" {
		return i18n.Lang(data.Invoice.Language)
	}
	return i18n.GetLang()
}
