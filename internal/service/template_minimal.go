package service

import (
	"fmt"

	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

var minGray = &props.Color{Red: 100, Green: 100, Blue: 100}
var minLightGray = &props.Color{Red: 150, Green: 150, Blue: 150}
var minLineColor = &props.Color{Red: 220, Green: 220, Blue: 220}
var minDivider = &props.Color{Red: 200, Green: 200, Blue: 200}

type MinimalTemplate struct{}

func (t *MinimalTemplate) Margins() TemplateMargins {
	return TemplateMargins{Left: 25, Top: 25, Right: 25}
}

func (t *MinimalTemplate) Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	m.AddRows(t.header(data)...)
	m.AddRow(20)
	m.AddRows(t.parties(data)...)
	m.AddRow(15)
	m.AddRows(t.payment(data)...)
	m.AddRow(15)
	m.AddRows(t.items(data)...)
	m.AddRow(10)
	m.AddRows(t.totals(data)...)

	if opts.ShowQR && opts.QRType != "none" {
		m.AddRow(15)
		m.AddRows(t.qr(data, opts)...)
	}
}

func (t *MinimalTemplate) header(data *InvoiceData) []core.Row {
	var rows []core.Row
	issueDate := data.Invoice.IssueDate.Format("02.01.2006")

	rows = append(rows, row.New(12).Add(
		text.NewCol(6, fmt.Sprintf("Faktura %s", data.Invoice.InvoiceNumber), props.Text{
			Size: 18, Style: fontstyle.Bold,
		}),
		text.NewCol(6, issueDate, props.Text{
			Size: 12, Align: align.Right, Top: 4, Color: minGray,
		}),
	))
	return rows
}

func (t *MinimalTemplate) parties(data *InvoiceData) []core.Row {
	var rows []core.Row

	rows = append(rows, row.New(5).Add(
		text.NewCol(12, data.Supplier.Name, props.Text{Size: 10, Style: fontstyle.Bold}),
	))
	rows = append(rows, row.New(4).Add(
		text.NewCol(12, fmt.Sprintf("%s, %s %s | IČO: %s",
			data.Supplier.Street, data.Supplier.ZIP, data.Supplier.City, data.Supplier.ICO),
			props.Text{Size: 9, Color: minGray}),
	))

	rows = append(rows, row.New(8))

	rows = append(rows, row.New(5).Add(
		text.NewCol(12, i18n.T("pdf.issued_for")+":", props.Text{Size: 8, Color: minLightGray}),
	))

	rows = append(rows, row.New(5).Add(
		text.NewCol(12, data.Customer.Name, props.Text{Size: 10, Style: fontstyle.Bold}),
	))
	rows = append(rows, row.New(4).Add(
		text.NewCol(12, fmt.Sprintf("%s, %s %s | IČO: %s",
			data.Customer.Street, data.Customer.ZIP, data.Customer.City, data.Customer.ICO),
			props.Text{Size: 9, Color: minGray}),
	))

	return rows
}

func (t *MinimalTemplate) payment(data *InvoiceData) []core.Row {
	var rows []core.Row
	dueDate := data.Invoice.DueDate.Format("02.01.2006")

	rows = append(rows, row.New(1).Add(line.NewCol(12, props.Line{Color: minLineColor})))

	rows = append(rows, row.New(8).Add(
		text.NewCol(3, fmt.Sprintf("%s: %s", i18n.T("pdf.due_date"), dueDate), props.Text{Size: 9, Style: fontstyle.Bold, Top: 2}),
		text.NewCol(3, fmt.Sprintf("VS: %s", data.Invoice.VariableSymbol), props.Text{Size: 9, Top: 2}),
		text.NewCol(6, fmt.Sprintf("%s: %s", i18n.T("pdf.bank_account"), data.BankAccount.AccountNumber), props.Text{Size: 9, Align: align.Right, Top: 2}),
	))

	rows = append(rows, row.New(1).Add(line.NewCol(12, props.Line{Color: minLineColor})))
	return rows
}

func (t *MinimalTemplate) items(data *InvoiceData) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	for _, item := range data.Items {
		rows = append(rows, row.New(7).Add(
			text.NewCol(8, item.Description, props.Text{Size: 10}),
			text.NewCol(2, fmt.Sprintf("%.0f x %.0f", item.Quantity, item.UnitPrice), props.Text{
				Size: 9, Align: align.Right, Color: minGray,
			}),
			text.NewCol(2, formatSimple(item.Total, currency), props.Text{
				Size: 10, Align: align.Right,
			}),
		))
	}

	return rows
}

func (t *MinimalTemplate) totals(data *InvoiceData) []core.Row {
	var rows []core.Row
	currency := data.Invoice.Currency

	rows = append(rows, row.New(1).Add(line.NewCol(12, props.Line{Color: minDivider})))

	rows = append(rows, row.New(12).Add(
		col.New(8),
		text.NewCol(2, i18n.T("pdf.total"), props.Text{Size: 12, Align: align.Right, Top: 3}),
		text.NewCol(2, formatSimple(data.Invoice.Total, currency), props.Text{
			Size: 14, Style: fontstyle.Bold, Align: align.Right, Top: 2,
		}),
	))

	if !data.Supplier.IsVATPayer {
		rows = append(rows, row.New(5).Add(
			text.NewCol(12, i18n.T("pdf.not_vat_payer"), props.Text{
				Size: 8, Align: align.Right, Color: minLightGray,
			}),
		))
	}

	return rows
}

func (t *MinimalTemplate) qr(data *InvoiceData, opts *TemplateOptions) []core.Row {
	var rows []core.Row

	spayd := GenerateQRPayload(opts.QRType, data)
	if spayd == "" {
		return rows
	}

	rows = append(rows, row.New(25).Add(
		col.New(9),
		code.NewQrCol(3, spayd, props.Rect{Percent: 100}),
	))
	return rows
}
