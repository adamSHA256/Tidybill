package service

import (
	"fmt"
	"os"

	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

var steelBlue = &props.Color{Red: 70, Green: 130, Blue: 180}
var grayText = &props.Color{Red: 80, Green: 80, Blue: 80}
var lightGray = &props.Color{Red: 100, Green: 100, Blue: 100}
var labelGray = &props.Color{Red: 120, Green: 120, Blue: 120}
var redAccent = &props.Color{Red: 200, Green: 50, Blue: 50}

type ModernTemplate struct{}

func (t *ModernTemplate) Margins() TemplateMargins {
	return TemplateMargins{Left: 20, Top: 20, Right: 20}
}

func (t *ModernTemplate) Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	lang := invoiceLang(data)
	m.AddRows(t.header(data, opts, lang)...)
	m.AddRow(15)
	m.AddRows(t.parties(data, lang)...)
	m.AddRow(15)
	m.AddRows(t.meta(data, opts, lang)...)
	m.AddRow(15)
	m.AddRows(t.items(data, lang)...)
	m.AddRow(10)
	m.AddRows(t.totals(data, lang)...)
	m.AddRow(15)
	m.AddRows(t.footer(data, opts, lang)...)
}

func (t *ModernTemplate) header(data *InvoiceData, opts *TemplateOptions, lang i18n.Lang) []core.Row {
	var rows []core.Row
	darkGray := &props.Color{Red: 50, Green: 50, Blue: 50}
	title := i18n.TfForLang(lang, "pdf.invoice_title", data.Invoice.InvoiceNumber)

	if opts.ShowLogo && data.Supplier.LogoPath != "" {
		if _, err := os.Stat(data.Supplier.LogoPath); err == nil {
			rows = append(rows, row.New(20).Add(
				image.NewFromFileCol(4, data.Supplier.LogoPath, props.Rect{Percent: 90}),
				col.New(4),
				col.New(4).Add(
					text.New(title, props.Text{Size: 28, Style: fontstyle.Bold, Align: align.Right, Color: darkGray}),
				),
			))
			rows = append(rows, row.New(10).Add(
				col.New(8),
				text.NewCol(4, data.Invoice.InvoiceNumber, props.Text{Size: 16, Align: align.Right, Color: lightGray}),
			))
			return rows
		}
	}

	rows = append(rows, row.New(20).Add(
		col.New(6).Add(
			text.New(title, props.Text{Size: 28, Style: fontstyle.Bold, Color: darkGray}),
		),
		col.New(6).Add(
			text.New(data.Invoice.InvoiceNumber, props.Text{Size: 16, Align: align.Right, Top: 8, Color: lightGray}),
		),
	))
	return rows
}

func (t *ModernTemplate) parties(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row

	rows = append(rows, row.New(8).Add(
		text.NewCol(5, i18n.TForLang(lang, "pdf.from"), props.Text{Size: 10, Style: fontstyle.Bold, Color: steelBlue}),
		col.New(2),
		text.NewCol(5, i18n.TForLang(lang, "pdf.for"), props.Text{Size: 10, Style: fontstyle.Bold, Color: steelBlue}),
	))

	rows = append(rows, row.New(7).Add(
		text.NewCol(5, data.Supplier.Name, props.Text{Size: 12, Style: fontstyle.Bold}),
		col.New(2),
		text.NewCol(5, data.Customer.Name, props.Text{Size: 12, Style: fontstyle.Bold}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(5, data.Supplier.Street, props.Text{Size: 9, Color: grayText}),
		col.New(2),
		text.NewCol(5, data.Customer.Street, props.Text{Size: 9, Color: grayText}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(5, fmt.Sprintf("%s %s", data.Supplier.ZIP, data.Supplier.City), props.Text{Size: 9, Color: grayText}),
		col.New(2),
		text.NewCol(5, fmt.Sprintf("%s %s", data.Customer.ZIP, data.Customer.City), props.Text{Size: 9, Color: grayText}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(5, i18n.TfForLang(lang, "pdf.ico", data.Supplier.ICO), props.Text{Size: 9}),
		col.New(2),
		text.NewCol(5, i18n.TfForLang(lang, "pdf.ico", data.Customer.ICO), props.Text{Size: 9}),
	))

	if data.Supplier.DIC != "" || data.Customer.DIC != "" {
		sDIC, cDIC := "", ""
		if data.Supplier.DIC != "" {
			sDIC = i18n.TfForLang(lang, "pdf.dic", data.Supplier.DIC)
		}
		if data.Customer.DIC != "" {
			cDIC = i18n.TfForLang(lang, "pdf.dic", data.Customer.DIC)
		}
		rows = append(rows, row.New(5).Add(
			text.NewCol(5, sDIC, props.Text{Size: 9}),
			col.New(2),
			text.NewCol(5, cDIC, props.Text{Size: 9}),
		))
	}

	if data.Supplier.ICDPH != "" || data.Customer.ICDPH != "" {
		sICDPH, cICDPH := "", ""
		if data.Supplier.ICDPH != "" {
			sICDPH = i18n.TfForLang(lang, "pdf.ic_dph", data.Supplier.ICDPH)
		}
		if data.Customer.ICDPH != "" {
			cICDPH = i18n.TfForLang(lang, "pdf.ic_dph", data.Customer.ICDPH)
		}
		rows = append(rows, row.New(5).Add(
			text.NewCol(5, sICDPH, props.Text{Size: 9}),
			col.New(2),
			text.NewCol(5, cICDPH, props.Text{Size: 9}),
		))
	}

	return rows
}

func (t *ModernTemplate) meta(data *InvoiceData, opts *TemplateOptions, lang i18n.Lang) []core.Row {
	var rows []core.Row
	issueDate := data.Invoice.IssueDate.Format("02.01.2006")
	dueDate := data.Invoice.DueDate.Format("02.01.2006")

	if opts.HasBankInfo {
		rows = append(rows, row.New(6).Add(
			col.New(3).Add(text.New(i18n.TForLang(lang, "pdf.issue_date"), props.Text{Size: 8, Color: labelGray})),
			col.New(3).Add(text.New(i18n.TForLang(lang, "pdf.due_date"), props.Text{Size: 8, Color: labelGray})),
			col.New(3).Add(text.New(i18n.TForLang(lang, "pdf.variable_symbol"), props.Text{Size: 8, Color: labelGray})),
			col.New(3).Add(text.New(i18n.TForLang(lang, "pdf.payment_method"), props.Text{Size: 8, Color: labelGray})),
		))

		rows = append(rows, row.New(6).Add(
			text.NewCol(3, issueDate, props.Text{Size: 10, Style: fontstyle.Bold}),
			text.NewCol(3, dueDate, props.Text{Size: 10, Style: fontstyle.Bold, Color: redAccent}),
			text.NewCol(3, data.Invoice.VariableSymbol, props.Text{Size: 10, Style: fontstyle.Bold}),
			text.NewCol(3, data.Invoice.PaymentMethod, props.Text{Size: 10}),
		))

		rows = append(rows, row.New(8))
		rows = append(rows, row.New(6).Add(
			col.New(3).Add(text.New(i18n.TForLang(lang, "pdf.bank_account"), props.Text{Size: 8, Color: labelGray})),
			col.New(9).Add(text.New("IBAN", props.Text{Size: 8, Color: labelGray})),
		))
		rows = append(rows, row.New(6).Add(
			text.NewCol(3, data.BankAccount.AccountNumber, props.Text{Size: 10, Style: fontstyle.Bold}),
			text.NewCol(9, data.BankAccount.IBAN, props.Text{Size: 10}),
		))
	} else {
		rows = append(rows, row.New(6).Add(
			col.New(4).Add(text.New(i18n.TForLang(lang, "pdf.issue_date"), props.Text{Size: 8, Color: labelGray})),
			col.New(4).Add(text.New(i18n.TForLang(lang, "pdf.due_date"), props.Text{Size: 8, Color: labelGray})),
			col.New(4).Add(text.New(i18n.TForLang(lang, "pdf.payment_method"), props.Text{Size: 8, Color: labelGray})),
		))

		rows = append(rows, row.New(6).Add(
			text.NewCol(4, issueDate, props.Text{Size: 10, Style: fontstyle.Bold}),
			text.NewCol(4, dueDate, props.Text{Size: 10, Style: fontstyle.Bold, Color: redAccent}),
			text.NewCol(4, data.Invoice.PaymentMethod, props.Text{Size: 10}),
		))
	}

	return rows
}

func (t *ModernTemplate) items(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	rows = append(rows, row.New(10).Add(
		text.NewCol(5, i18n.TForLang(lang, "pdf.col_description"), props.Text{Size: 9, Style: fontstyle.Bold, Color: steelBlue}),
		text.NewCol(2, i18n.TForLang(lang, "pdf.col_quantity"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: steelBlue}),
		text.NewCol(2, i18n.TForLang(lang, "pdf.col_unit_price"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: steelBlue}),
		text.NewCol(1, i18n.TForLang(lang, "pdf.col_vat"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: steelBlue}),
		text.NewCol(2, i18n.TForLang(lang, "pdf.col_total"), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: steelBlue}),
	))

	for _, item := range data.Items {
		rows = append(rows, row.New(8).Add(
			text.NewCol(5, item.Description, props.Text{Size: 10}),
			text.NewCol(2, fmt.Sprintf("%.0f %s", item.Quantity, item.Unit), props.Text{Size: 10, Align: align.Right}),
			text.NewCol(2, formatSimple(item.UnitPrice, currency), props.Text{Size: 10, Align: align.Right}),
			text.NewCol(1, fmt.Sprintf("%.0f%%", item.VATRate), props.Text{Size: 10, Align: align.Right}),
			text.NewCol(2, formatSimple(item.Total, currency), props.Text{Size: 10, Align: align.Right, Style: fontstyle.Bold}),
		))
	}

	return rows
}

func (t *ModernTemplate) totals(data *InvoiceData, lang i18n.Lang) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	rows = append(rows, row.New(7).Add(
		col.New(8),
		text.NewCol(2, i18n.TForLang(lang, "pdf.subtotal"), props.Text{Size: 10, Align: align.Right, Color: lightGray}),
		text.NewCol(2, formatSimple(data.Invoice.Subtotal, currency), props.Text{Size: 10, Align: align.Right}),
	))

	if data.Invoice.VATTotal > 0 {
		rows = append(rows, row.New(7).Add(
			col.New(8),
			text.NewCol(2, i18n.TForLang(lang, "pdf.vat_total"), props.Text{Size: 10, Align: align.Right, Color: lightGray}),
			text.NewCol(2, formatSimple(data.Invoice.VATTotal, currency), props.Text{Size: 10, Align: align.Right}),
		))
	}

	rows = append(rows, row.New(12).Add(
		col.New(8),
		text.NewCol(2, i18n.TForLang(lang, "pdf.total"), props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Right, Top: 3}),
		text.NewCol(2, formatSimple(data.Invoice.Total, currency), props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Right, Top: 3, Color: steelBlue}),
	))

	return rows
}

func (t *ModernTemplate) footer(data *InvoiceData, opts *TemplateOptions, lang i18n.Lang) []core.Row {
	var rows []core.Row

	if opts.ShowQR && opts.QRType != "none" {
		spayd := GenerateQRPayload(opts.QRType, data)
		if spayd != "" {
			rows = append(rows, row.New(8).Add(
				text.NewCol(12, i18n.TForLang(lang, "pdf.qr_payment"), props.Text{Size: 9, Style: fontstyle.Bold, Color: steelBlue}),
			))
			notesText := ""
			if opts.ShowNotes {
				notesText = data.Invoice.Notes
			}
			rows = append(rows, row.New(30).Add(
				code.NewQrCol(3, spayd, props.Rect{Percent: 100}),
				col.New(1),
				col.New(8).Add(
					text.New(notesText, props.Text{Size: 9, Color: grayText}),
				),
			))
			return rows
		}
	}

	if opts.ShowNotes && data.Invoice.Notes != "" {
		rows = append(rows, row.New(20).Add(
			text.NewCol(12, data.Invoice.Notes, props.Text{Size: 9, Color: grayText}),
		))
	}

	return rows
}
