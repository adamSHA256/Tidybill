package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/user/invoice-app/internal/model"
)

type PDFService struct {
	pdfDir string
}

func NewPDFService(pdfDir string) *PDFService {
	return &PDFService{pdfDir: pdfDir}
}

type InvoiceData struct {
	Invoice     *model.Invoice
	Supplier    *model.Supplier
	Customer    *model.Customer
	BankAccount *model.BankAccount
	Items       []model.InvoiceItem
}

func (s *PDFService) GenerateInvoice(data *InvoiceData) (string, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(10).
		WithRightMargin(10).
		WithTopMargin(10).
		Build()

	m := maroto.New(cfg)

	// Header
	s.addHeader(m, data)

	// Supplier and Customer info
	s.addParties(m, data)

	// Payment info bar
	s.addPaymentBar(m, data)

	// Items table
	s.addItemsTable(m, data)

	// VAT summary and totals
	s.addTotals(m, data)

	// QR code and final info
	s.addFooter(m, data)

	// Generate PDF
	doc, err := m.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Save to file
	year := data.Invoice.IssueDate.Year()
	yearDir := filepath.Join(s.pdfDir, fmt.Sprintf("%d", year))
	if err := os.MkdirAll(yearDir, 0755); err != nil {
		return "", err
	}

	pdfPath := filepath.Join(yearDir, data.Invoice.InvoiceNumber+".pdf")
	if err := doc.Save(pdfPath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return pdfPath, nil
}

func (s *PDFService) addHeader(m core.Maroto, data *InvoiceData) {
	m.AddRow(15,
		col.New(12).Add(
			text.New(fmt.Sprintf("Faktura - daňový doklad %s", data.Invoice.InvoiceNumber),
				props.Text{
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
		),
	)
	m.AddRow(5,
		col.New(12).Add(
			line.New(props.Line{Color: &props.Color{Red: 0, Green: 0, Blue: 0}}),
		),
	)
}

func (s *PDFService) addParties(m core.Maroto, data *InvoiceData) {
	m.AddRow(5)

	m.AddRow(80,
		// Supplier column
		col.New(4).Add(
			text.New("DODAVATEL", props.Text{Size: 10, Style: fontstyle.Bold}),
			text.New(data.Supplier.Name, props.Text{Size: 10, Style: fontstyle.Bold, Top: 5}),
			text.New(data.Supplier.Street, props.Text{Size: 9, Top: 10}),
			text.New(fmt.Sprintf("%s %s", data.Supplier.ZIP, data.Supplier.City), props.Text{Size: 9, Top: 14}),
			text.New(fmt.Sprintf("IČO: %s", data.Supplier.ICO), props.Text{Size: 9, Top: 22}),
			text.New(fmt.Sprintf("DIČ: %s", data.Supplier.DIC), props.Text{Size: 9, Top: 26}),
			text.New(fmt.Sprintf("Tel: %s", data.Supplier.Phone), props.Text{Size: 9, Top: 34}),
			text.New(fmt.Sprintf("E-mail: %s", data.Supplier.Email), props.Text{Size: 9, Top: 38}),
			text.New(s.vatPayerText(data.Supplier.IsVATPayer), props.Text{Size: 8, Style: fontstyle.Italic, Top: 48}),
		),

		// Customer column
		col.New(4).Add(
			text.New("ODBĚRATEL", props.Text{Size: 10, Style: fontstyle.Bold}),
			text.New(data.Customer.Name, props.Text{Size: 10, Style: fontstyle.Bold, Top: 5}),
			text.New(data.Customer.Street, props.Text{Size: 9, Top: 10}),
			text.New(fmt.Sprintf("%s %s", data.Customer.ZIP, data.Customer.City), props.Text{Size: 9, Top: 14}),
			text.New(fmt.Sprintf("Země: %s", data.Customer.Country), props.Text{Size: 9, Top: 18}),
			text.New(fmt.Sprintf("IČO: %s", data.Customer.ICO), props.Text{Size: 9, Top: 26}),
			text.New(fmt.Sprintf("DIČ: %s", data.Customer.DIC), props.Text{Size: 9, Top: 30}),
		),

		// Payment info column
		col.New(4).Add(
			text.New("Datum vystavení:", props.Text{Size: 9}),
			text.New(data.Invoice.IssueDate.Format("02.01.2006"), props.Text{Size: 9, Style: fontstyle.Bold, Left: 35}),
			text.New("Datum splatnosti:", props.Text{Size: 9, Top: 6}),
			text.New(data.Invoice.DueDate.Format("02.01.2006"), props.Text{Size: 9, Style: fontstyle.Bold, Top: 6, Left: 35}),
			text.New("Forma úhrady:", props.Text{Size: 9, Top: 12}),
			text.New("Převodem", props.Text{Size: 9, Style: fontstyle.Bold, Top: 12, Left: 35}),
			text.New("Bankovní účet:", props.Text{Size: 9, Top: 20}),
			text.New(data.BankAccount.AccountNumber, props.Text{Size: 9, Style: fontstyle.Bold, Top: 20, Left: 35}),
			text.New("IBAN:", props.Text{Size: 9, Top: 26}),
			text.New(data.BankAccount.IBAN, props.Text{Size: 8, Style: fontstyle.Bold, Top: 26, Left: 35}),
			text.New("Variabilní symbol:", props.Text{Size: 9, Top: 34}),
			text.New(data.Invoice.VariableSymbol, props.Text{Size: 9, Style: fontstyle.Bold, Top: 34, Left: 35}),
		),
	)
}

func (s *PDFService) addPaymentBar(m core.Maroto, data *InvoiceData) {
	grayBg := &props.Color{Red: 240, Green: 240, Blue: 240}

	m.AddRow(15,
		col.New(4).Add(
			text.New("Variabilní symbol", props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(data.Invoice.VariableSymbol, props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),

		col.New(4).Add(
			text.New("Datum splatnosti", props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(data.Invoice.DueDate.Format("02.01.2006"), props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),

		col.New(4).Add(
			text.New("Suma k úhradě", props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(s.formatMoney(data.Invoice.Total, data.Invoice.Currency), props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
	)

	m.AddRow(5)
}

func (s *PDFService) addItemsTable(m core.Maroto, data *InvoiceData) {
	// Header
	headerColor := &props.Color{Red: 220, Green: 220, Blue: 220}
	headerStyle := props.Text{Size: 8, Style: fontstyle.Bold}

	m.AddRow(8,
		col.New(5).Add(text.New("Popis", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New("Počet", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(2).Add(text.New("Cena/ks", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New("DPH %", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New("DPH", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(2).Add(text.New("Celkem", headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
	)

	// Items
	cellStyle := props.Text{Size: 8}
	numStyle := props.Text{Size: 8, Align: align.Right}

	for _, item := range data.Items {
		m.AddRow(7,
			col.New(5).Add(text.New(item.Description, cellStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
			col.New(1).Add(text.New(fmt.Sprintf("%.0f", item.Quantity), numStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
			col.New(2).Add(text.New(s.formatMoney(item.UnitPrice, ""), numStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
			col.New(1).Add(text.New(fmt.Sprintf("%.0f", item.VATRate), numStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
			col.New(1).Add(text.New(s.formatMoney(item.VATAmount, ""), numStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
			col.New(2).Add(text.New(s.formatMoney(item.Total, ""), numStyle)).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
		)
	}

	m.AddRow(5)
}

func (s *PDFService) addTotals(m core.Maroto, data *InvoiceData) {
	rightStyle := props.Text{Size: 9, Align: align.Right}
	rightBold := props.Text{Size: 10, Align: align.Right, Style: fontstyle.Bold}

	m.AddRow(6,
		col.New(8),
		col.New(2).Add(text.New("Základ:", rightStyle)),
		col.New(2).Add(text.New(s.formatMoney(data.Invoice.Subtotal, data.Invoice.Currency), rightStyle)),
	)
	m.AddRow(6,
		col.New(8),
		col.New(2).Add(text.New("DPH:", rightStyle)),
		col.New(2).Add(text.New(s.formatMoney(data.Invoice.VATTotal, data.Invoice.Currency), rightStyle)),
	)
	m.AddRow(2,
		col.New(8),
		col.New(4).Add(line.New(props.Line{Color: &props.Color{Red: 0, Green: 0, Blue: 0}})),
	)
	m.AddRow(8,
		col.New(8),
		col.New(2).Add(text.New("CELKEM:", rightBold)),
		col.New(2).Add(text.New(s.formatMoney(data.Invoice.Total, data.Invoice.Currency), rightBold)),
	)

	m.AddRow(10)
}

func (s *PDFService) addFooter(m core.Maroto, data *InvoiceData) {
	// Generate SPAYD QR code
	spayd := s.generateSPAYD(data)

	m.AddRow(40,
		col.New(3).Add(
			code.NewQr(spayd, props.Rect{
				Percent: 100,
				Center:  true,
			}),
		),
		col.New(9).Add(
			text.New("QR kód pro platbu", props.Text{Size: 8, Style: fontstyle.Italic}),
		),
	)
}

func (s *PDFService) generateSPAYD(data *InvoiceData) string {
	// SPAYD format for Czech banking
	return fmt.Sprintf("SPD*1.0*ACC:%s*AM:%.2f*CC:%s*X-VS:%s*MSG:Faktura %s",
		data.BankAccount.IBAN,
		data.Invoice.Total,
		data.Invoice.Currency,
		data.Invoice.VariableSymbol,
		data.Invoice.InvoiceNumber,
	)
}

func (s *PDFService) formatMoney(amount float64, currency string) string {
	// Czech format: 1 234,56 Kč
	intPart := int(amount)
	decPart := int((amount - float64(intPart)) * 100)

	formatted := fmt.Sprintf("%d,%02d", intPart, decPart)

	// Add thousand separators
	if intPart >= 1000 {
		str := fmt.Sprintf("%d", intPart)
		var result string
		for i, c := range str {
			if i > 0 && (len(str)-i)%3 == 0 {
				result += " "
			}
			result += string(c)
		}
		formatted = fmt.Sprintf("%s,%02d", result, decPart)
	}

	if currency != "" {
		formatted += " " + currency
	}
	return formatted
}

func (s *PDFService) vatPayerText(isVATPayer bool) string {
	if isVATPayer {
		return "Plátce DPH"
	}
	return "Nejsem plátce DPH"
}

// AddLogo adds logo to PDF if available
func (s *PDFService) addLogo(m core.Maroto, logoPath string) {
	if logoPath == "" {
		return
	}
	if _, err := os.Stat(logoPath); err != nil {
		return
	}

	m.AddRow(20,
		col.New(3).Add(
			image.NewFromFile(logoPath, props.Rect{
				Percent: 80,
				Center:  true,
			}),
		),
		col.New(9),
	)
}
