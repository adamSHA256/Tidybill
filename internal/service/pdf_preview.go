package service

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
)

//go:embed assets/default_logo.jpg
var defaultLogoFS embed.FS

// getDefaultLogoPath extracts the embedded logo to a temp file and returns its path.
func getDefaultLogoPath() string {
	data, err := defaultLogoFS.ReadFile("assets/default_logo.jpg")
	if err != nil {
		return ""
	}
	tmp := filepath.Join(os.TempDir(), "tidybill_default_logo.jpg")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return ""
	}
	return tmp
}

// GeneratePreview generates a preview PDF with sample data for a given template
func (s *PDFService) GeneratePreview(templateCode string, opts *TemplateOptions, lang i18n.Lang) (string, error) {
	return s.GeneratePreviewWithYAML(templateCode, "", opts, lang)
}

// GeneratePreviewWithYAML generates a preview PDF, optionally using YAML source for custom templates.
func (s *PDFService) GeneratePreviewWithYAML(templateCode, yamlSource string, opts *TemplateOptions, lang i18n.Lang) (string, error) {
	sampleData := buildSampleInvoiceData(lang)

	if err := os.MkdirAll(s.previewDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create preview dir: %w", err)
	}

	previewPath := filepath.Join(s.previewDir, templateCode+"_preview.pdf")
	return s.generateToPathWithYAML(sampleData, templateCode, yamlSource, opts, previewPath)
}

// GenerateAllPreviews generates preview PDFs for ALL templates
func (s *PDFService) GenerateAllPreviews(templates []*model.PDFTemplate, lang i18n.Lang) (map[string]string, error) {
	results := make(map[string]string)
	for _, t := range templates {
		opts := &TemplateOptions{
			ShowLogo:    t.ShowLogo,
			ShowQR:      t.ShowQR,
			ShowNotes:   t.ShowNotes,
			QRType:      "spayd",
			HasBankInfo: true,
		}
		yamlSource := t.YAMLSource
		if t.IsBuiltin {
			yamlSource = ""
		}
		path, err := s.GeneratePreviewWithYAML(t.TemplateCode, yamlSource, opts, lang)
		if err != nil {
			results[t.ID] = "error: " + err.Error()
			continue
		}
		results[t.ID] = path
	}
	return results, nil
}

// generateToPath generates a PDF to a specific file path
func (s *PDFService) generateToPath(data *InvoiceData, templateCode string, opts *TemplateOptions, outputPath string) (string, error) {
	return s.generateToPathWithYAML(data, templateCode, "", opts, outputPath)
}

// generateToPathWithYAML generates a PDF, optionally using YAML source for custom templates.
func (s *PDFService) generateToPathWithYAML(data *InvoiceData, templateCode, yamlSource string, opts *TemplateOptions, outputPath string) (string, error) {
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

	if err := doc.Save(outputPath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return outputPath, nil
}

func buildSampleInvoiceData(lang i18n.Lang) *InvoiceData {
	now := time.Now()

	logoPath := getDefaultLogoPath()

	return &InvoiceData{
		Invoice: &model.Invoice{
			InvoiceNumber:  "VF99-00042",
			IssueDate:      now,
			DueDate:        now.AddDate(0, 0, 14),
			TaxableDate:    now,
			VariableSymbol: "9900042",
			PaymentMethod:  i18n.TForLang(lang, "payment_type.bank_transfer"),
			Currency:       "CZK",
			Subtotal:       79600.00,
			VATTotal:       0,
			Total:          79600.00,
			Notes:          i18n.TForLang(lang, "preview.notes"),
			Language:       string(lang),
			TemplateID:     "table",
		},
		Supplier: &model.Supplier{
			Name:       i18n.TForLang(lang, "preview.supplier_name"),
			Street:     "Prazska 123/4",
			City:       "Praha 1",
			ZIP:        "110 00",
			Country:    "CZ",
			ICO:        "12345678",
			DIC:        "CZ12345678",
			Phone:      "+420 123 456 789",
			Email:      "info@ukazka.cz",
			IsVATPayer: false,
			LogoPath:   logoPath,
		},
		Customer: &model.Customer{
			Name:    i18n.TForLang(lang, "preview.customer_name"),
			Street:  "Brnenska 567",
			City:    "Brno",
			ZIP:     "602 00",
			Country: "CZ",
			ICO:     "87654321",
			DIC:     "CZ87654321",
		},
		BankAccount: &model.BankAccount{
			AccountNumber: "123456789/0100",
			IBAN:          "CZ6501000000000123456789",
			Currency:      "CZK",
			QRType:        "spayd",
		},
		Items: []model.InvoiceItem{
			{Description: i18n.TForLang(lang, "preview.item1"), Quantity: 40, Unit: "hod", UnitPrice: 1500, VATRate: 0, Subtotal: 60000, Total: 60000},
			{Description: i18n.TForLang(lang, "preview.item2"), Quantity: 8, Unit: "hod", UnitPrice: 2000, VATRate: 0, Subtotal: 16000, Total: 16000},
			{Description: i18n.TForLang(lang, "preview.item3"), Quantity: 1, Unit: "ks", UnitPrice: 3600, VATRate: 0, Subtotal: 3600, Total: 3600},
		},
	}
}
