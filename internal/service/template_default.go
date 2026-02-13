package service

import (
	"fmt"
	"os"

	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type DefaultTemplate struct{}

func (t *DefaultTemplate) Margins() TemplateMargins {
	return TemplateMargins{Left: 10, Top: 10, Right: 10}
}

func (t *DefaultTemplate) Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	t.addHeader(m, data, opts)
	t.addParties(m, data)
	t.addPaymentBar(m, data)
	t.addItemsTable(m, data)
	t.addTotals(m, data)
	t.addFooter(m, data, opts)
}

func (t *DefaultTemplate) addHeader(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	if opts.ShowLogo && data.Supplier.LogoPath != "" {
		if _, err := os.Stat(data.Supplier.LogoPath); err == nil {
			m.AddRow(20,
				col.New(3).Add(
					image.NewFromFile(data.Supplier.LogoPath, props.Rect{
						Percent: 80,
						Center:  true,
					}),
				),
				col.New(9).Add(
					text.New(i18n.Tf("pdf.invoice_title", data.Invoice.InvoiceNumber),
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
			return
		}
	}

	m.AddRow(15,
		col.New(12).Add(
			text.New(i18n.Tf("pdf.invoice_title", data.Invoice.InvoiceNumber),
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

func (t *DefaultTemplate) addParties(m core.Maroto, data *InvoiceData) {
	m.AddRow(5)
	m.AddRow(80,
		col.New(4).Add(
			text.New(i18n.T("pdf.supplier"), props.Text{Size: 10, Style: fontstyle.Bold}),
			text.New(data.Supplier.Name, props.Text{Size: 10, Style: fontstyle.Bold, Top: 5}),
			text.New(data.Supplier.Street, props.Text{Size: 9, Top: 10}),
			text.New(fmt.Sprintf("%s %s", data.Supplier.ZIP, data.Supplier.City), props.Text{Size: 9, Top: 14}),
			text.New(i18n.Tf("pdf.ico", data.Supplier.ICO), props.Text{Size: 9, Top: 22}),
			text.New(i18n.Tf("pdf.dic", data.Supplier.DIC), props.Text{Size: 9, Top: 26}),
			text.New(i18n.Tf("pdf.phone", data.Supplier.Phone), props.Text{Size: 9, Top: 34}),
			text.New(i18n.Tf("pdf.email", data.Supplier.Email), props.Text{Size: 9, Top: 38}),
			text.New(t.vatPayerText(data.Supplier.IsVATPayer), props.Text{Size: 8, Style: fontstyle.Italic, Top: 48}),
		),
		col.New(4).Add(
			text.New(i18n.T("pdf.customer"), props.Text{Size: 10, Style: fontstyle.Bold}),
			text.New(data.Customer.Name, props.Text{Size: 10, Style: fontstyle.Bold, Top: 5}),
			text.New(data.Customer.Street, props.Text{Size: 9, Top: 10}),
			text.New(fmt.Sprintf("%s %s", data.Customer.ZIP, data.Customer.City), props.Text{Size: 9, Top: 14}),
			text.New(i18n.Tf("pdf.country", data.Customer.Country), props.Text{Size: 9, Top: 18}),
			text.New(i18n.Tf("pdf.ico", data.Customer.ICO), props.Text{Size: 9, Top: 26}),
			text.New(i18n.Tf("pdf.dic", data.Customer.DIC), props.Text{Size: 9, Top: 30}),
		),
		col.New(4).Add(
			text.New(i18n.T("pdf.issue_date"), props.Text{Size: 9}),
			text.New(data.Invoice.IssueDate.Format("02.01.2006"), props.Text{Size: 9, Style: fontstyle.Bold, Left: 35}),
			text.New(i18n.T("pdf.due_date"), props.Text{Size: 9, Top: 6}),
			text.New(data.Invoice.DueDate.Format("02.01.2006"), props.Text{Size: 9, Style: fontstyle.Bold, Top: 6, Left: 35}),
			text.New(i18n.T("pdf.payment_method"), props.Text{Size: 9, Top: 12}),
			text.New(i18n.T("pdf.bank_transfer"), props.Text{Size: 9, Style: fontstyle.Bold, Top: 12, Left: 35}),
			text.New(i18n.T("pdf.bank_account"), props.Text{Size: 9, Top: 20}),
			text.New(data.BankAccount.AccountNumber, props.Text{Size: 9, Style: fontstyle.Bold, Top: 20, Left: 35}),
			text.New(i18n.T("pdf.iban"), props.Text{Size: 9, Top: 26}),
			text.New(data.BankAccount.IBAN, props.Text{Size: 8, Style: fontstyle.Bold, Top: 26, Left: 35}),
			text.New(i18n.T("pdf.variable_symbol"), props.Text{Size: 9, Top: 34}),
			text.New(data.Invoice.VariableSymbol, props.Text{Size: 9, Style: fontstyle.Bold, Top: 34, Left: 35}),
		),
	)
}

func (t *DefaultTemplate) addPaymentBar(m core.Maroto, data *InvoiceData) {
	grayBg := &props.Color{Red: 240, Green: 240, Blue: 240}

	m.AddRow(15,
		col.New(4).Add(
			text.New(i18n.T("pdf.variable_symbol"), props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(data.Invoice.VariableSymbol, props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
		col.New(4).Add(
			text.New(i18n.T("pdf.due_date"), props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(data.Invoice.DueDate.Format("02.01.2006"), props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
		col.New(4).Add(
			text.New(i18n.T("pdf.amount_due"), props.Text{Size: 8, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}),
			text.New(formatMoneyDefault(data.Invoice.Total, data.Invoice.Currency), props.Text{Size: 10, Style: fontstyle.Bold, Top: 4}),
		).WithStyle(&props.Cell{BackgroundColor: grayBg, BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}),
	)
	m.AddRow(5)
}

func (t *DefaultTemplate) addItemsTable(m core.Maroto, data *InvoiceData) {
	headerColor := &props.Color{Red: 220, Green: 220, Blue: 220}
	headerStyle := props.Text{Size: 8, Style: fontstyle.Bold}

	m.AddRow(8,
		col.New(5).Add(text.New(i18n.T("pdf.col_description"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New(i18n.T("pdf.col_quantity"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(2).Add(text.New(i18n.T("pdf.col_unit_price"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New(i18n.T("pdf.col_vat_rate"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(1).Add(text.New(i18n.T("pdf.col_vat"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
		col.New(2).Add(text.New(i18n.T("pdf.col_total"), headerStyle)).WithStyle(&props.Cell{BackgroundColor: headerColor, BorderType: border.Full}),
	)

	cellStyle := props.Text{Size: 8}
	numStyle := props.Text{Size: 8, Align: align.Right}
	borderStyle := &props.Cell{BorderType: border.Full, BorderColor: &props.Color{Red: 200, Green: 200, Blue: 200}}

	for _, item := range data.Items {
		m.AddRow(7,
			col.New(5).Add(text.New(item.Description, cellStyle)).WithStyle(borderStyle),
			col.New(1).Add(text.New(fmt.Sprintf("%.0f", item.Quantity), numStyle)).WithStyle(borderStyle),
			col.New(2).Add(text.New(formatMoneyDefault(item.UnitPrice, ""), numStyle)).WithStyle(borderStyle),
			col.New(1).Add(text.New(fmt.Sprintf("%.0f", item.VATRate), numStyle)).WithStyle(borderStyle),
			col.New(1).Add(text.New(formatMoneyDefault(item.VATAmount, ""), numStyle)).WithStyle(borderStyle),
			col.New(2).Add(text.New(formatMoneyDefault(item.Total, ""), numStyle)).WithStyle(borderStyle),
		)
	}
	m.AddRow(5)
}

func (t *DefaultTemplate) addTotals(m core.Maroto, data *InvoiceData) {
	rightStyle := props.Text{Size: 9, Align: align.Right}
	rightBold := props.Text{Size: 10, Align: align.Right, Style: fontstyle.Bold}

	m.AddRow(6,
		col.New(8),
		col.New(2).Add(text.New(i18n.T("pdf.subtotal"), rightStyle)),
		col.New(2).Add(text.New(formatMoneyDefault(data.Invoice.Subtotal, data.Invoice.Currency), rightStyle)),
	)
	m.AddRow(6,
		col.New(8),
		col.New(2).Add(text.New(i18n.T("pdf.vat_total"), rightStyle)),
		col.New(2).Add(text.New(formatMoneyDefault(data.Invoice.VATTotal, data.Invoice.Currency), rightStyle)),
	)
	m.AddRow(2,
		col.New(8),
		col.New(4).Add(line.New(props.Line{Color: &props.Color{Red: 0, Green: 0, Blue: 0}})),
	)
	m.AddRow(8,
		col.New(8),
		col.New(2).Add(text.New(i18n.T("pdf.total"), rightBold)),
		col.New(2).Add(text.New(formatMoneyDefault(data.Invoice.Total, data.Invoice.Currency), rightBold)),
	)
	m.AddRow(10)
}

func (t *DefaultTemplate) addFooter(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	if opts.ShowQR && opts.QRType != "none" {
		spayd := GenerateQRPayload(opts.QRType, data)
		if spayd != "" {
			m.AddRow(40,
				col.New(3).Add(
					code.NewQr(spayd, props.Rect{
						Percent: 100,
						Center:  true,
					}),
				),
				col.New(9).Add(
					text.New(i18n.T("pdf.qr_payment"), props.Text{Size: 8, Style: fontstyle.Italic}),
				),
			)
		}
	}

	if opts.ShowNotes && data.Invoice.Notes != "" {
		m.AddRow(15,
			col.New(12).Add(
				text.New(data.Invoice.Notes, props.Text{Size: 9}),
			),
		)
	}
}

func (t *DefaultTemplate) vatPayerText(isVATPayer bool) string {
	if isVATPayer {
		return i18n.T("pdf.vat_payer")
	}
	return i18n.T("pdf.not_vat_payer")
}

// formatMoneyDefault formats money in Czech format (1 234,56 Kč)
func formatMoneyDefault(amount float64, currency string) string {
	intPart := int(amount)
	decPart := int((amount - float64(intPart)) * 100)

	formatted := fmt.Sprintf("%d,%02d", intPart, decPart)

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
