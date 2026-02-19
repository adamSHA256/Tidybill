package service

import (
	"fmt"
	"os"

	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

const (
	classicPadLeft  = 2.0
	classicPadRight = 2.0
)

type ClassicTemplate struct{}

func (t *ClassicTemplate) Margins() TemplateMargins {
	return TemplateMargins{Left: 15, Top: 15, Right: 15}
}

func (t *ClassicTemplate) Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	lang := invoiceLang(data)
	m.AddRows(t.header(data, opts, lang)...)
	m.AddRow(5)
	m.AddRow(1, line.NewCol(12))
	m.AddRow(5)
	m.AddRows(t.parties(data, lang)...)
	m.AddRow(8)
	m.AddRow(1, line.NewCol(12))
	m.AddRow(5)
	m.AddRows(t.details(data, opts, lang)...)
	m.AddRow(8)
	m.AddRows(t.items(data, lang)...)
	m.AddRow(5)
	m.AddRows(t.totals(data, lang)...)
	m.AddRow(10)
	m.AddRows(t.footer(data, opts)...)
}

func (t *ClassicTemplate) header(data *InvoiceData, opts *TemplateOptions, lang i18n.Lang) []core.Row {
	var rows []core.Row
	title := i18n.TfForLang(lang, "pdf.invoice_title", data.Invoice.InvoiceNumber)

	if opts.ShowLogo && data.Supplier.LogoPath != "" {
		if _, err := os.Stat(data.Supplier.LogoPath); err == nil {
			rows = append(rows, row.New(25).Add(
				image.NewFromFileCol(3, data.Supplier.LogoPath, props.Rect{
					Percent: 80,
					Left:    classicPadLeft,
				}),
				col.New(9).Add(
					text.New(title, props.Text{
						Size: 20, Style: fontstyle.Bold, Align: align.Center, Top: 5,
					}),
				),
			))
			return rows
		}
	}

	rows = append(rows, row.New(15).Add(
		text.NewCol(12, title, props.Text{
			Size: 20, Style: fontstyle.Bold, Align: align.Center,
			Left: classicPadLeft, Right: classicPadRight,
		}),
	))
	return rows
}

func (t *ClassicTemplate) parties(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row

	rows = append(rows, row.New(7).Add(
		text.NewCol(4, i18n.TForLang(lang, "pdf.supplier")+":", props.Text{Size: 11, Style: fontstyle.Bold, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, i18n.TForLang(lang, "pdf.customer")+":", props.Text{Size: 11, Style: fontstyle.Bold, Right: classicPadRight}),
	))

	rows = append(rows, row.New(6).Add(
		text.NewCol(4, data.Supplier.Name, props.Text{Size: 10, Style: fontstyle.Bold, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, data.Customer.Name, props.Text{Size: 10, Style: fontstyle.Bold, Right: classicPadRight}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(4, data.Supplier.Street, props.Text{Size: 9, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, data.Customer.Street, props.Text{Size: 9, Right: classicPadRight}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(4, fmt.Sprintf("%s %s", data.Supplier.ZIP, data.Supplier.City), props.Text{Size: 9, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, fmt.Sprintf("%s %s", data.Customer.ZIP, data.Customer.City), props.Text{Size: 9, Right: classicPadRight}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(4, i18n.TfForLang(lang, "pdf.ico", data.Supplier.ICO), props.Text{Size: 9, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, i18n.TfForLang(lang, "pdf.ico", data.Customer.ICO), props.Text{Size: 9, Right: classicPadRight}),
	))

	supplierDIC := i18n.TForLang(lang, "pdf.not_vat_payer")
	if data.Supplier.DIC != "" {
		supplierDIC = i18n.TfForLang(lang, "pdf.dic", data.Supplier.DIC)
	}
	customerDIC := ""
	if data.Customer.DIC != "" {
		customerDIC = i18n.TfForLang(lang, "pdf.dic", data.Customer.DIC)
	}
	rows = append(rows, row.New(5).Add(
		text.NewCol(4, supplierDIC, props.Text{Size: 9, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, customerDIC, props.Text{Size: 9, Right: classicPadRight}),
	))

	return rows
}

func (t *ClassicTemplate) details(data *InvoiceData, opts *TemplateOptions, lang i18n.Lang) []core.Row {
	var rows []core.Row
	issueDate := data.Invoice.IssueDate.Format("02.01.2006")
	dueDate := data.Invoice.DueDate.Format("02.01.2006")
	taxDate := data.Invoice.TaxableDate.Format("02.01.2006")

	rows = append(rows, row.New(7).Add(
		text.NewCol(4, i18n.TForLang(lang, "pdf.payment_info")+":", props.Text{Size: 11, Style: fontstyle.Bold, Left: classicPadLeft}),
		col.New(4),
		text.NewCol(4, i18n.TForLang(lang, "pdf.dates")+":", props.Text{Size: 11, Style: fontstyle.Bold, Right: classicPadRight}),
	))

	if opts.HasBankInfo {
		rows = append(rows, row.New(5).Add(
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.bank_account"), data.BankAccount.AccountNumber), props.Text{Size: 9, Left: classicPadLeft}),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.issue_date"), issueDate), props.Text{Size: 9, Right: classicPadRight}),
		))

		rows = append(rows, row.New(5).Add(
			text.NewCol(4, fmt.Sprintf("IBAN: %s", data.BankAccount.IBAN), props.Text{Size: 9, Left: classicPadLeft}),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.due_date"), dueDate), props.Text{Size: 9, Style: fontstyle.Bold, Right: classicPadRight}),
		))

		rows = append(rows, row.New(5).Add(
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.variable_symbol"), data.Invoice.VariableSymbol), props.Text{Size: 9, Style: fontstyle.Bold, Left: classicPadLeft}),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.taxable_date"), taxDate), props.Text{Size: 9, Right: classicPadRight}),
		))
	} else {
		rows = append(rows, row.New(5).Add(
			col.New(4),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.issue_date"), issueDate), props.Text{Size: 9, Right: classicPadRight}),
		))

		rows = append(rows, row.New(5).Add(
			col.New(4),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.due_date"), dueDate), props.Text{Size: 9, Style: fontstyle.Bold, Right: classicPadRight}),
		))

		rows = append(rows, row.New(5).Add(
			col.New(4),
			col.New(4),
			text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.taxable_date"), taxDate), props.Text{Size: 9, Right: classicPadRight}),
		))
	}

	rows = append(rows, row.New(5).Add(
		text.NewCol(4, fmt.Sprintf("%s: %s", i18n.TForLang(lang, "pdf.payment_method"), data.Invoice.PaymentMethod), props.Text{Size: 9, Left: classicPadLeft}),
		col.New(8),
	))

	return rows
}

func (t *ClassicTemplate) items(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	rows = append(rows, row.New(8).Add(
		text.NewCol(5, i18n.TForLang(lang, "pdf.col_description"), props.Text{Size: 9, Style: fontstyle.Bold, Left: classicPadLeft}),
		text.NewCol(1, i18n.TForLang(lang, "pdf.col_quantity"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(1, i18n.TForLang(lang, "pdf.col_unit"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Center}),
		text.NewCol(2, i18n.TForLang(lang, "pdf.col_unit_price"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(1, i18n.TForLang(lang, "pdf.col_vat_rate"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, i18n.TForLang(lang, "pdf.col_total"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Right: classicPadRight}),
	))

	rows = append(rows, row.New(1).Add(line.NewCol(12)))

	for _, item := range data.Items {
		rows = append(rows, row.New(6).Add(
			text.NewCol(5, item.Description, props.Text{Size: 9, Left: classicPadLeft}),
			text.NewCol(1, fmt.Sprintf("%.0f", item.Quantity), props.Text{Size: 9, Align: align.Right}),
			text.NewCol(1, item.Unit, props.Text{Size: 9, Align: align.Center}),
			text.NewCol(2, formatSimple(item.UnitPrice, currency), props.Text{Size: 9, Align: align.Right}),
			text.NewCol(1, fmt.Sprintf("%.0f%%", item.VATRate), props.Text{Size: 9, Align: align.Right}),
			text.NewCol(2, formatSimple(item.Total, currency), props.Text{Size: 9, Align: align.Right, Right: classicPadRight}),
		))
	}

	rows = append(rows, row.New(1).Add(line.NewCol(12)))
	return rows
}

func (t *ClassicTemplate) totals(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	rows = append(rows, row.New(6).Add(
		col.New(8),
		text.NewCol(2, i18n.TForLang(lang, "pdf.subtotal")+":", props.Text{Size: 10, Align: align.Right}),
		text.NewCol(2, formatSimple(data.Invoice.Subtotal, currency), props.Text{Size: 10, Align: align.Right, Right: classicPadRight}),
	))

	if data.Invoice.VATTotal > 0 {
		rows = append(rows, row.New(6).Add(
			col.New(8),
			text.NewCol(2, i18n.TForLang(lang, "pdf.vat_total")+":", props.Text{Size: 10, Align: align.Right}),
			text.NewCol(2, formatSimple(data.Invoice.VATTotal, currency), props.Text{Size: 10, Align: align.Right, Right: classicPadRight}),
		))
	}

	rows = append(rows, row.New(8).Add(
		col.New(8),
		text.NewCol(2, i18n.TForLang(lang, "pdf.total")+":", props.Text{Size: 12, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, formatSimple(data.Invoice.Total, currency), props.Text{Size: 12, Style: fontstyle.Bold, Align: align.Right, Right: classicPadRight}),
	))

	return rows
}

func (t *ClassicTemplate) footer(data *InvoiceData, opts *TemplateOptions) []core.Row {
	var rows []core.Row

	if opts.ShowQR && opts.QRType != "none" {
		spayd := GenerateQRPayload(opts.QRType, data)
		if spayd != "" {
			notesText := ""
			if opts.ShowNotes {
				notesText = data.Invoice.Notes
			}
			rows = append(rows, row.New(35).Add(
				code.NewQrCol(3, spayd, props.Rect{
					Percent: 80,
					Left:    classicPadLeft,
				}),
				col.New(1),
				text.NewCol(8, notesText, props.Text{
					Size: 9, Top: 5, Right: classicPadRight,
				}),
			))
			return rows
		}
	}

	if opts.ShowNotes && data.Invoice.Notes != "" {
		rows = append(rows, row.New(15).Add(
			text.NewCol(12, data.Invoice.Notes, props.Text{
				Size: 9, Left: classicPadLeft, Right: classicPadRight,
			}),
		))
	}

	return rows
}

// formatSimple formats money as "1234.56 CZK"
func formatSimple(amount float64, currency string) string {
	return fmt.Sprintf("%.2f %s", amount, currency)
}
