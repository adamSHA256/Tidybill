package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/model"
)

type PDFService struct {
	pdfDir     string
	previewDir string
	fonts      []*entity.CustomFont
}

func NewPDFService(pdfDir, previewDir string) *PDFService {
	return &PDFService{
		pdfDir:     pdfDir,
		previewDir: previewDir,
		fonts:      LoadEmbeddedFonts(),
	}
}

type InvoiceData struct {
	Invoice     *model.Invoice
	Supplier    *model.Supplier
	Customer    *model.Customer
	BankAccount *model.BankAccount
	Items       []model.InvoiceItem
}

func (s *PDFService) GenerateInvoice(data *InvoiceData, templateCode string, opts *TemplateOptions) (string, error) {
	return s.GenerateInvoiceWithYAML(data, templateCode, "", opts)
}

func (s *PDFService) GenerateInvoiceWithYAML(data *InvoiceData, templateCode, yamlSource string, opts *TemplateOptions) (string, error) {
	renderer := s.getRenderer(templateCode, yamlSource)
	margins := renderer.Margins()

	cfgBuilder := config.NewBuilder().
		WithLeftMargin(margins.Left).
		WithRightMargin(margins.Right).
		WithTopMargin(margins.Top)

	if len(s.fonts) > 0 {
		cfgBuilder = cfgBuilder.
			WithCustomFonts(s.fonts).
			WithDefaultFont(&props.Font{Family: FontFamily, Size: 10})
	}

	cfg := cfgBuilder.Build()
	m := maroto.New(cfg)

	renderer.Render(m, data, opts)

	doc, err := m.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate PDF: %w", err)
	}

	year := data.Invoice.IssueDate.Year()
	supplierDir := sanitizeDirName(data.Supplier.Name)
	yearDir := filepath.Join(s.pdfDir, supplierDir, fmt.Sprintf("%d", year))
	if err := os.MkdirAll(yearDir, 0755); err != nil {
		return "", err
	}

	pdfPath := filepath.Join(yearDir, data.Invoice.InvoiceNumber+".pdf")
	if err := doc.Save(pdfPath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return pdfPath, nil
}

// getRenderer returns the appropriate renderer for a template.
// Built-in templates use the compiled Go code; custom templates use the YAML interpreter.
func (s *PDFService) getRenderer(templateCode, yamlSource string) TemplateRenderer {
	// If YAML source is provided and this is not a built-in template, use declarative renderer
	if yamlSource != "" {
		if r, err := NewDeclarativeRenderer(yamlSource); err == nil {
			return r
		}
	}
	// Fall back to built-in registry
	return GetTemplateRenderer(templateCode)
}

func sanitizeDirName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	result := strings.TrimSpace(replacer.Replace(name))
	if result == "" {
		return "default"
	}
	return result
}
