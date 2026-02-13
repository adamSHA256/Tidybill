package service

import (
	"github.com/johnfercher/maroto/v2/pkg/core"
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
