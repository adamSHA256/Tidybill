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

//go:embed assets/default_logo.png
var defaultLogoFS embed.FS

// getDefaultLogoPath extracts the embedded logo to a temp file and returns its path.
func getDefaultLogoPath() string {
	data, err := defaultLogoFS.ReadFile("assets/default_logo.png")
	if err != nil {
		return ""
	}
	tmp := filepath.Join(os.TempDir(), "tidybill_default_logo.png")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return ""
	}
	return tmp
}

// GeneratePreview generates a preview PDF with sample data for a given template
func (s *PDFService) GeneratePreview(templateCode string, opts *TemplateOptions) (string, error) {
	return s.GeneratePreviewWithYAML(templateCode, "", opts)
}

// GeneratePreviewWithYAML generates a preview PDF, optionally using YAML source for custom templates.
func (s *PDFService) GeneratePreviewWithYAML(templateCode, yamlSource string, opts *TemplateOptions) (string, error) {
	sampleData := buildSampleInvoiceData()

	if err := os.MkdirAll(s.previewDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create preview dir: %w", err)
	}

	previewPath := filepath.Join(s.previewDir, templateCode+"_preview.pdf")
	return s.generateToPathWithYAML(sampleData, templateCode, yamlSource, opts, previewPath)
}

// GenerateAllPreviews generates preview PDFs for ALL templates
func (s *PDFService) GenerateAllPreviews(templates []*model.PDFTemplate) (map[string]string, error) {
	results := make(map[string]string)
	for _, t := range templates {
		opts := &TemplateOptions{
			ShowLogo:  t.ShowLogo,
			ShowQR:    t.ShowQR,
			ShowNotes: t.ShowNotes,
			QRType:    "spayd",
		}
		yamlSource := t.YAMLSource
		if t.IsBuiltin {
			yamlSource = ""
		}
		path, err := s.GeneratePreviewWithYAML(t.TemplateCode, yamlSource, opts)
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

func buildSampleInvoiceData() *InvoiceData {
	now := time.Now()

	logoPath := getDefaultLogoPath()

	return &InvoiceData{
		Invoice: &model.Invoice{
			InvoiceNumber:  "VF99-00042",
			IssueDate:      now,
			DueDate:        now.AddDate(0, 0, 14),
			TaxableDate:    now,
			VariableSymbol: "9900042",
			PaymentMethod:  i18n.T("payment_type.bank_transfer"),
			Currency:       "CZK",
			Subtotal:       79600.00,
			VATTotal:       0,
			Total:          79600.00,
			Notes:          "Toto je ukazkovy dokument. Dekujeme za Vasi duveru a tesime se na dalsi spolupraci.",
			TemplateID:     "default",
		},
		Supplier: &model.Supplier{
			Name:       "Ukazkova Firma s.r.o.",
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
			Name:    "Testovaci Zakaznik a.s.",
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
			{Description: "Vyvoj webove aplikace", Quantity: 40, Unit: "hod", UnitPrice: 1500, VATRate: 0, Subtotal: 60000, Total: 60000},
			{Description: "Konzultace a analyza pozadavku", Quantity: 8, Unit: "hod", UnitPrice: 2000, VATRate: 0, Subtotal: 16000, Total: 16000},
			{Description: "Sprava serveru (mesicni)", Quantity: 1, Unit: "ks", UnitPrice: 3600, VATRate: 0, Subtotal: 3600, Total: 3600},
		},
	}
}
