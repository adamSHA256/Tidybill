package service

// BuiltinYAMLTemplates maps template codes to their YAML equivalents.
// These are used as the starting point when a user duplicates a built-in template.
var BuiltinYAMLTemplates = map[string]string{
	"table": yamlDefault,
	"classic": yamlClassic,
	"modern":  yamlModern,
	"minimal": yamlMinimal,
}

// GetBuiltinYAML returns the YAML source for a built-in template code.
func GetBuiltinYAML(code string) string {
	if y, ok := BuiltinYAMLTemplates[code]; ok {
		return y
	}
	return ""
}

var yamlDefault = `name: "Tabulková"
margins: { left: 10, top: 10, right: 10 }

colors:
  black: { r: 0, g: 0, b: 0 }
  gray: { r: 100, g: 100, b: 100 }
  light_border: { r: 200, g: 200, b: 200 }
  header_bg: { r: 240, g: 240, b: 240 }
  item_header_bg: { r: 220, g: 220, b: 220 }

layout:
  # ── Header with logo ──
  - if: "and .Options.ShowLogo .Supplier.LogoPath"
    row: 20
    cols:
      - width: 3
        logo: true
      - width: 6
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 16, bold: true, align: center }
      - width: 3

  - if: "and .Options.ShowLogo .Supplier.LogoPath"
    row: 5
    cols:
      - width: 12
        text: ""

  # ── Header without logo ──
  - if: "not (and .Options.ShowLogo .Supplier.LogoPath)"
    row: 15
    cols:
      - width: 12
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 16, bold: true, align: center }

  - line: { color: black, size_percent: 100 }
  - spacer: 5

  # ── Parties (3-column layout) ──
  - row: 80
    cols:
      - width: 4
        texts:
          - text: "{{ label \"pdf.supplier\" }}"
            style: { size: 10, bold: true }
          - text: "{{ .Supplier.Name }}"
            style: { size: 10, bold: true, top: 5 }
          - text: "{{ .Supplier.Street }}"
            style: { size: 9, top: 10 }
          - text: "{{ .Supplier.ZIP }} {{ .Supplier.City }}"
            style: { size: 9, top: 14 }
          - text: "{{ labelf \"pdf.ico\" .Supplier.ICO }}"
            style: { size: 9, top: 22 }
          - text: "{{ labelf \"pdf.dic\" .Supplier.DIC }}"
            style: { size: 9, top: 26 }
          - text: "{{ if .Supplier.ICDPH }}{{ labelf \"pdf.ic_dph\" .Supplier.ICDPH }}{{ end }}"
            style: { size: 9, top: 30 }
          - text: "{{ labelf \"pdf.phone\" .Supplier.Phone }}"
            style: { size: 9, top: 38 }
          - text: "{{ labelf \"pdf.email\" .Supplier.Email }}"
            style: { size: 9, top: 42 }
          - text: "{{ if .Supplier.IsVATPayer }}{{ label \"pdf.vat_payer\" }}{{ else }}{{ label \"pdf.not_vat_payer\" }}{{ end }}"
            style: { size: 8, italic: true, top: 52 }
      - width: 4
        texts:
          - text: "{{ label \"pdf.customer\" }}"
            style: { size: 10, bold: true }
          - text: "{{ .Customer.Name }}"
            style: { size: 10, bold: true, top: 5 }
          - text: "{{ .Customer.Street }}"
            style: { size: 9, top: 10 }
          - text: "{{ .Customer.ZIP }} {{ .Customer.City }}"
            style: { size: 9, top: 14 }
          - text: "{{ labelf \"pdf.country\" .Customer.Country }}"
            style: { size: 9, top: 18 }
          - text: "{{ labelf \"pdf.ico\" .Customer.ICO }}"
            style: { size: 9, top: 26 }
          - text: "{{ labelf \"pdf.dic\" .Customer.DIC }}"
            style: { size: 9, top: 30 }
          - text: "{{ if .Customer.ICDPH }}{{ labelf \"pdf.ic_dph\" .Customer.ICDPH }}{{ end }}"
            style: { size: 9, top: 34 }
      - width: 4
        texts:
          - text: "{{ label \"pdf.issue_date\" }}"
            style: { size: 9 }
          - text: "{{ .Invoice.IssueDate | date }}"
            style: { size: 9, bold: true, left: 35 }
          - text: "{{ label \"pdf.due_date\" }}"
            style: { size: 9, top: 6 }
          - text: "{{ .Invoice.DueDate | date }}"
            style: { size: 9, bold: true, top: 6, left: 35 }
          - text: "{{ if .Supplier.IsVATPayer }}{{ label \"pdf.taxable_date\" }}{{ end }}"
            style: { size: 9, top: 12 }
          - text: "{{ if .Supplier.IsVATPayer }}{{ .Invoice.TaxableDate | date }}{{ end }}"
            style: { size: 9, bold: true, top: 12, left: 35 }
          - text: "{{ label \"pdf.payment_method\" }}"
            style: { size: 9, top: 18 }
          - text: "{{ .Invoice.PaymentMethod }}"
            style: { size: 9, bold: true, top: 18, left: 35 }
          - text: "{{ if .Options.HasBankInfo }}{{ label \"pdf.bank_account\" }}{{ end }}"
            style: { size: 9, top: 26 }
          - text: "{{ if .Options.HasBankInfo }}{{ .BankAccount.AccountNumber }}{{ end }}"
            style: { size: 9, bold: true, top: 26, left: 35 }
          - text: "{{ if .Options.HasBankInfo }}{{ label \"pdf.iban\" }}{{ end }}"
            style: { size: 9, top: 32 }
          - text: "{{ if .Options.HasBankInfo }}{{ .BankAccount.IBAN }}{{ end }}"
            style: { size: 8, bold: true, top: 32, left: 35 }
          - text: "{{ if .Options.HasBankInfo }}{{ label \"pdf.variable_symbol\" }}{{ end }}"
            style: { size: 9, top: 40 }
          - text: "{{ if .Options.HasBankInfo }}{{ .Invoice.VariableSymbol }}{{ end }}"
            style: { size: 9, bold: true, top: 40, left: 35 }

  # ── Payment bar ──
  - row: 15
    cols:
      - width: 4
        texts:
          - text: "{{ if .Options.HasBankInfo }}{{ label \"pdf.variable_symbol\" }}{{ else }}{{ label \"pdf.payment_method\" }}{{ end }}"
            style: { size: 8, color: gray }
          - text: "{{ if .Options.HasBankInfo }}{{ .Invoice.VariableSymbol }}{{ else }}{{ .Invoice.PaymentMethod }}{{ end }}"
            style: { size: 10, bold: true, top: 4 }
        cell: { border: full, bg_color: header_bg, border_color: light_border }
      - width: 4
        texts:
          - text: "{{ label \"pdf.due_date\" }}"
            style: { size: 8, color: gray }
          - text: "{{ .Invoice.DueDate | date }}"
            style: { size: 10, bold: true, top: 4 }
        cell: { border: full, bg_color: header_bg, border_color: light_border }
      - width: 4
        texts:
          - text: "{{ label \"pdf.amount_due\" }}"
            style: { size: 8, color: gray }
          - text: "{{ .Invoice.Total | money }}"
            style: { size: 10, bold: true, top: 4 }
        cell: { border: full, bg_color: header_bg, border_color: light_border }

  - spacer: 5

  # ── Items table ──
  - items:
      row_height: 7
      header_cell: { border: full, bg_color: item_header_bg }
      row_cell: { border: full, border_color: light_border }
      header:
        - width: 5
          text: "{{ label \"pdf.col_description\" }}"
          style: { size: 8, bold: true }
        - width: 1
          text: "{{ label \"pdf.col_quantity\" }}"
          style: { size: 8, bold: true }
        - width: 2
          text: "{{ label \"pdf.col_unit_price\" }}"
          style: { size: 8, bold: true }
        - width: 1
          text: "{{ label \"pdf.col_vat_rate\" }}"
          style: { size: 8, bold: true }
        - width: 1
          text: "{{ label \"pdf.col_vat\" }}"
          style: { size: 8, bold: true }
        - width: 2
          text: "{{ label \"pdf.col_total\" }}"
          style: { size: 8, bold: true }
      cols:
        - width: 5
          field: Description
          style: { size: 8 }
        - width: 1
          field: Quantity
          style: { size: 8, align: right }
        - width: 2
          field: UnitPrice
          format: money
          style: { size: 8, align: right }
        - width: 1
          field: VATRate
          style: { size: 8, align: right }
        - width: 1
          field: VATAmount
          format: money
          style: { size: 8, align: right }
        - width: 2
          field: Total
          format: money
          style: { size: 8, align: right }

  - spacer: 5

  # ── Totals ──
  - row: 6
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.subtotal\" }}"
        style: { size: 9, align: right }
      - width: 2
        text: "{{ .Invoice.Subtotal | money }}"
        style: { size: 9, align: right }

  - row: 6
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.vat_total\" }}"
        style: { size: 9, align: right }
      - width: 2
        text: "{{ .Invoice.VATTotal | money }}"
        style: { size: 9, align: right }

  - row: 2
    cols:
      - width: 8
      - width: 4
        text: ""

  - row: 8
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.total\" }}"
        style: { size: 10, bold: true, align: right }
      - width: 2
        text: "{{ .Invoice.Total | money }}"
        style: { size: 10, bold: true, align: right }

  - spacer: 10

  # ── Footer: QR ──
  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    row: 40
    cols:
      - width: 3
        qr: true
      - width: 9
        text: "{{ label \"pdf.qr_payment\" }}"
        style: { size: 8, italic: true }

  # ── Footer: Notes ──
  - notes: true
`

var yamlClassic = `name: "Klasická"
margins: { left: 15, top: 15, right: 15 }

colors:
  black: { r: 0, g: 0, b: 0 }

layout:
  # ── Header with logo ──
  - if: "and .Options.ShowLogo .Supplier.LogoPath"
    row: 25
    cols:
      - width: 3
        logo: true
      - width: 6
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 20, bold: true, align: center, top: 5 }
      - width: 3

  # ── Header without logo ──
  - if: "not (and .Options.ShowLogo .Supplier.LogoPath)"
    row: 15
    cols:
      - width: 12
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 20, bold: true, align: center, left: 2, right: 2 }

  - spacer: 5
  - line: { color: black, size_percent: 100 }
  - spacer: 5

  # ── Parties ──
  - row: 7
    cols:
      - width: 4
        text: "{{ label \"pdf.supplier\" }}:"
        style: { size: 11, bold: true, left: 2 }
      - width: 4
      - width: 4
        text: "{{ label \"pdf.customer\" }}:"
        style: { size: 11, bold: true, right: 2 }

  - row: 6
    cols:
      - width: 4
        text: "{{ .Supplier.Name }}"
        style: { size: 10, bold: true, left: 2 }
      - width: 4
      - width: 4
        text: "{{ .Customer.Name }}"
        style: { size: 10, bold: true, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ .Supplier.Street }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ .Customer.Street }}"
        style: { size: 9, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ .Supplier.ZIP }} {{ .Supplier.City }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ .Customer.ZIP }} {{ .Customer.City }}"
        style: { size: 9, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ labelf \"pdf.ico\" .Supplier.ICO }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ labelf \"pdf.ico\" .Customer.ICO }}"
        style: { size: 9, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ labelf \"pdf.dic\" .Supplier.DIC }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ labelf \"pdf.dic\" .Customer.DIC }}"
        style: { size: 9, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ if .Supplier.ICDPH }}{{ labelf \"pdf.ic_dph\" .Supplier.ICDPH }}{{ end }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ if .Customer.ICDPH }}{{ labelf \"pdf.ic_dph\" .Customer.ICDPH }}{{ end }}"
        style: { size: 9, right: 2 }

  - if: "not .Supplier.IsVATPayer"
    row: 5
    cols:
      - width: 4
        text: "{{ label \"pdf.not_vat_payer\" }}"
        style: { size: 8, italic: true, left: 2 }
      - width: 8

  - spacer: 8
  - line: { color: black, size_percent: 100 }
  - spacer: 5

  # ── Payment details ──
  - row: 7
    cols:
      - width: 4
        text: "{{ label \"pdf.payment_info\" }}:"
        style: { size: 11, bold: true, left: 2 }
      - width: 4
      - width: 4
        text: "{{ label \"pdf.dates\" }}:"
        style: { size: 11, bold: true, right: 2 }

  - if: ".Options.HasBankInfo"
    row: 5
    cols:
      - width: 4
        text: "{{ label \"pdf.bank_account\" }} {{ .BankAccount.AccountNumber }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ label \"pdf.issue_date\" }} {{ .Invoice.IssueDate | date }}"
        style: { size: 9, right: 2 }

  - if: ".Options.HasBankInfo"
    row: 5
    cols:
      - width: 4
        text: "IBAN: {{ .BankAccount.IBAN }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ label \"pdf.due_date\" }} {{ .Invoice.DueDate | date }}"
        style: { size: 9, bold: true, right: 2 }

  - if: ".Options.HasBankInfo"
    row: 5
    cols:
      - width: 4
        text: "{{ label \"pdf.variable_symbol\" }} {{ .Invoice.VariableSymbol }}"
        style: { size: 9, bold: true, left: 2 }
      - width: 4
      - width: 4
        text: "{{ if .Supplier.IsVATPayer }}{{ label \"pdf.taxable_date\" }}: {{ .Invoice.TaxableDate | date }}{{ end }}"
        style: { size: 9, right: 2 }

  - if: "not .Options.HasBankInfo"
    row: 5
    cols:
      - width: 4
        text: "{{ label \"pdf.payment_method\" }} {{ .Invoice.PaymentMethod }}"
        style: { size: 9, left: 2 }
      - width: 4
      - width: 4
        text: "{{ label \"pdf.issue_date\" }} {{ .Invoice.IssueDate | date }}"
        style: { size: 9, right: 2 }

  - if: "not .Options.HasBankInfo"
    row: 5
    cols:
      - width: 4
      - width: 4
      - width: 4
        text: "{{ label \"pdf.due_date\" }} {{ .Invoice.DueDate | date }}"
        style: { size: 9, bold: true, right: 2 }

  - if: "and (not .Options.HasBankInfo) .Supplier.IsVATPayer"
    row: 5
    cols:
      - width: 4
      - width: 4
      - width: 4
        text: "{{ label \"pdf.taxable_date\" }}: {{ .Invoice.TaxableDate | date }}"
        style: { size: 9, right: 2 }

  - row: 5
    cols:
      - width: 4
        text: "{{ if .Options.HasBankInfo }}{{ label \"pdf.payment_method\" }} {{ .Invoice.PaymentMethod }}{{ end }}"
        style: { size: 9, left: 2 }
      - width: 8

  - spacer: 8

  # ── Items ──
  - items:
      row_height: 6
      header:
        - width: 5
          text: "{{ label \"pdf.col_description\" }}"
          style: { size: 9, bold: true, left: 2 }
        - width: 1
          text: "{{ label \"pdf.col_quantity\" }}"
          style: { size: 9, bold: true, align: right }
        - width: 1
          text: "{{ label \"pdf.col_unit\" }}"
          style: { size: 9, bold: true, align: center }
        - width: 2
          text: "{{ label \"pdf.col_unit_price\" }}"
          style: { size: 9, bold: true, align: right }
        - width: 1
          text: "{{ label \"pdf.col_vat_rate\" }}"
          style: { size: 9, bold: true, align: right }
        - width: 2
          text: "{{ label \"pdf.col_total\" }}"
          style: { size: 9, bold: true, align: right, right: 2 }
      cols:
        - width: 5
          field: Description
          style: { size: 9, left: 2 }
        - width: 1
          field: Quantity
          style: { size: 9, align: right }
        - width: 1
          field: Unit
          style: { size: 9, align: center }
        - width: 2
          field: UnitPrice
          format: money
          style: { size: 9, align: right }
        - width: 1
          field: VATRate
          style: { size: 9, align: right }
        - width: 2
          field: Total
          format: money
          style: { size: 9, align: right, right: 2 }

  - spacer: 5

  # ── Totals ──
  - row: 6
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.subtotal\" }}"
        style: { size: 10, align: right }
      - width: 2
        text: "{{ .Invoice.Subtotal | money }}"
        style: { size: 10, align: right, right: 2 }

  - if: "gt .Invoice.VATTotal 0"
    row: 6
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.vat_total\" }}"
        style: { size: 10, align: right }
      - width: 2
        text: "{{ .Invoice.VATTotal | money }}"
        style: { size: 10, align: right, right: 2 }

  - row: 8
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.total\" }}"
        style: { size: 12, bold: true, align: right }
      - width: 2
        text: "{{ .Invoice.Total | money }}"
        style: { size: 12, bold: true, align: right, right: 2 }

  - spacer: 10

  # ── Footer ──
  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    row: 35
    cols:
      - width: 3
        qr: true
      - width: 1
      - width: 8
        text: "{{ .Invoice.Notes }}"
        style: { size: 9, right: 2 }

  - notes: true
`

var yamlModern = `name: "Moderní"
margins: { left: 20, top: 20, right: 20 }

colors:
  steel_blue: { r: 70, g: 130, b: 180 }
  gray_text: { r: 80, g: 80, b: 80 }
  light_gray: { r: 100, g: 100, b: 100 }
  label_gray: { r: 120, g: 120, b: 120 }
  red_accent: { r: 200, g: 50, b: 50 }
  dark_gray: { r: 50, g: 50, b: 50 }

layout:
  # ── Header with logo ──
  - if: "and .Options.ShowLogo .Supplier.LogoPath"
    row: 20
    cols:
      - width: 4
        logo: true
      - width: 2
      - width: 6
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 16, align: right, top: 4, color: light_gray }

  # ── Header without logo ──
  - if: "not (and .Options.ShowLogo .Supplier.LogoPath)"
    row: 20
    cols:
      - width: 12
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 16, align: right, top: 4, color: light_gray }

  - spacer: 15

  # ── Parties ──
  - row: 8
    cols:
      - width: 5
        text: "{{ label \"pdf.from\" }}"
        style: { size: 10, bold: true, color: steel_blue }
      - width: 2
      - width: 5
        text: "{{ label \"pdf.for\" }}"
        style: { size: 10, bold: true, color: steel_blue }

  - row: 7
    cols:
      - width: 5
        text: "{{ .Supplier.Name }}"
        style: { size: 12, bold: true }
      - width: 2
      - width: 5
        text: "{{ .Customer.Name }}"
        style: { size: 12, bold: true }

  - row: 5
    cols:
      - width: 5
        text: "{{ .Supplier.Street }}"
        style: { size: 9, color: gray_text }
      - width: 2
      - width: 5
        text: "{{ .Customer.Street }}"
        style: { size: 9, color: gray_text }

  - row: 5
    cols:
      - width: 5
        text: "{{ .Supplier.ZIP }} {{ .Supplier.City }}"
        style: { size: 9, color: gray_text }
      - width: 2
      - width: 5
        text: "{{ .Customer.ZIP }} {{ .Customer.City }}"
        style: { size: 9, color: gray_text }

  - row: 5
    cols:
      - width: 5
        text: "{{ labelf \"pdf.ico\" .Supplier.ICO }}"
        style: { size: 9 }
      - width: 2
      - width: 5
        text: "{{ labelf \"pdf.ico\" .Customer.ICO }}"
        style: { size: 9 }

  - row: 5
    cols:
      - width: 5
        text: "{{ labelf \"pdf.dic\" .Supplier.DIC }}"
        style: { size: 9 }
      - width: 2
      - width: 5
        text: "{{ labelf \"pdf.dic\" .Customer.DIC }}"
        style: { size: 9 }

  - row: 5
    cols:
      - width: 5
        text: "{{ if .Supplier.ICDPH }}{{ labelf \"pdf.ic_dph\" .Supplier.ICDPH }}{{ end }}"
        style: { size: 9 }
      - width: 2
      - width: 5
        text: "{{ if .Customer.ICDPH }}{{ labelf \"pdf.ic_dph\" .Customer.ICDPH }}{{ end }}"
        style: { size: 9 }

  - if: "not .Supplier.IsVATPayer"
    row: 5
    cols:
      - width: 5
        text: "{{ label \"pdf.not_vat_payer\" }}"
        style: { size: 8, italic: true, color: gray_text }
      - width: 7

  - spacer: 15

  # ── Meta (with bank info) ──
  - if: ".Options.HasBankInfo"
    row: 6
    cols:
      - width: 3
        text: "{{ label \"pdf.issue_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 3
        text: "{{ label \"pdf.due_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 3
        text: "{{ label \"pdf.variable_symbol\" }}"
        style: { size: 8, color: label_gray }
      - width: 3
        text: "{{ label \"pdf.payment_method\" }}"
        style: { size: 8, color: label_gray }

  - if: ".Options.HasBankInfo"
    row: 6
    cols:
      - width: 3
        text: "{{ .Invoice.IssueDate | date }}"
        style: { size: 10, bold: true }
      - width: 3
        text: "{{ .Invoice.DueDate | date }}"
        style: { size: 10, bold: true, color: red_accent }
      - width: 3
        text: "{{ .Invoice.VariableSymbol }}"
        style: { size: 10, bold: true }
      - width: 3
        text: "{{ .Invoice.PaymentMethod }}"
        style: { size: 10 }

  - if: "and .Options.HasBankInfo .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 3
        text: "{{ label \"pdf.taxable_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 9

  - if: "and .Options.HasBankInfo .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 3
        text: "{{ .Invoice.TaxableDate | date }}"
        style: { size: 10 }
      - width: 9

  - if: ".Options.HasBankInfo"
    spacer: 8

  - if: ".Options.HasBankInfo"
    row: 6
    cols:
      - width: 3
        text: "{{ label \"pdf.bank_account\" }}"
        style: { size: 8, color: label_gray }
      - width: 9
        text: "IBAN"
        style: { size: 8, color: label_gray }

  - if: ".Options.HasBankInfo"
    row: 6
    cols:
      - width: 3
        text: "{{ .BankAccount.AccountNumber }}"
        style: { size: 10, bold: true }
      - width: 9
        text: "{{ .BankAccount.IBAN }}"
        style: { size: 10 }

  # ── Meta (without bank info) ──
  - if: "not .Options.HasBankInfo"
    row: 6
    cols:
      - width: 4
        text: "{{ label \"pdf.issue_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 4
        text: "{{ label \"pdf.due_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 4
        text: "{{ label \"pdf.payment_method\" }}"
        style: { size: 8, color: label_gray }

  - if: "not .Options.HasBankInfo"
    row: 6
    cols:
      - width: 4
        text: "{{ .Invoice.IssueDate | date }}"
        style: { size: 10, bold: true }
      - width: 4
        text: "{{ .Invoice.DueDate | date }}"
        style: { size: 10, bold: true, color: red_accent }
      - width: 4
        text: "{{ .Invoice.PaymentMethod }}"
        style: { size: 10 }

  - if: "and (not .Options.HasBankInfo) .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 4
        text: "{{ label \"pdf.taxable_date\" }}"
        style: { size: 8, color: label_gray }
      - width: 8

  - if: "and (not .Options.HasBankInfo) .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 4
        text: "{{ .Invoice.TaxableDate | date }}"
        style: { size: 10 }
      - width: 8

  - spacer: 15

  # ── Items ──
  - items:
      row_height: 8
      header:
        - width: 5
          text: "{{ label \"pdf.col_description\" }}"
          style: { size: 9, bold: true, color: steel_blue }
        - width: 2
          text: "{{ label \"pdf.col_quantity\" }}"
          style: { size: 9, bold: true, align: right, color: steel_blue }
        - width: 2
          text: "{{ label \"pdf.col_unit_price\" }}"
          style: { size: 9, bold: true, align: right, color: steel_blue }
        - width: 1
          text: "{{ label \"pdf.col_vat\" }}"
          style: { size: 9, bold: true, align: right, color: steel_blue }
        - width: 2
          text: "{{ label \"pdf.col_total\" }}"
          style: { size: 9, bold: true, align: right, color: steel_blue }
      cols:
        - width: 5
          field: Description
          style: { size: 10 }
        - width: 2
          text: "{{ .Quantity | num0 }} {{ .Unit }}"
          style: { size: 10, align: right }
        - width: 2
          field: UnitPrice
          format: money
          style: { size: 10, align: right }
        - width: 1
          field: VATRate
          style: { size: 10, align: right }
        - width: 2
          field: Total
          format: money
          style: { size: 10, align: right, bold: true }

  - spacer: 10

  # ── Totals ──
  - row: 7
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.subtotal\" }}"
        style: { size: 10, align: right, color: light_gray }
      - width: 2
        text: "{{ .Invoice.Subtotal | money }}"
        style: { size: 10, align: right }

  - if: "gt .Invoice.VATTotal 0"
    row: 7
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.vat_total\" }}"
        style: { size: 10, align: right, color: light_gray }
      - width: 2
        text: "{{ .Invoice.VATTotal | money }}"
        style: { size: 10, align: right }

  - row: 12
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.total\" }}"
        style: { size: 14, bold: true, align: right, top: 3 }
      - width: 2
        text: "{{ .Invoice.Total | money }}"
        style: { size: 14, bold: true, align: right, top: 3, color: steel_blue }

  - spacer: 15

  # ── Footer ──
  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    row: 8
    cols:
      - width: 12
        text: "{{ label \"pdf.qr_payment\" }}"
        style: { size: 9, bold: true, color: steel_blue }

  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    row: 30
    cols:
      - width: 3
        qr: true
      - width: 1
      - width: 8
        text: "{{ .Invoice.Notes }}"
        style: { size: 9, color: gray_text }

  - notes: true
`

var yamlMinimal = `name: "Minimální"
margins: { left: 25, top: 25, right: 25 }

colors:
  gray: { r: 100, g: 100, b: 100 }
  light_gray: { r: 150, g: 150, b: 150 }
  line_color: { r: 220, g: 220, b: 220 }
  divider: { r: 200, g: 200, b: 200 }

layout:
  # ── Header ──
  - row: 12
    cols:
      - width: 6
        text: "{{ invoiceTitle .Invoice.InvoiceNumber .Supplier.IsVATPayer }}"
        style: { size: 18, bold: true }
      - width: 6
        text: "{{ .Invoice.IssueDate | date }}"
        style: { size: 12, align: right, top: 4, color: gray }

  - spacer: 20

  # ── Parties ──
  - row: 5
    cols:
      - width: 12
        text: "{{ .Supplier.Name }}"
        style: { size: 10, bold: true }

  - row: 4
    cols:
      - width: 12
        text: "{{ .Supplier.Street }}, {{ .Supplier.ZIP }} {{ .Supplier.City }} | {{ labelf \"pdf.ico\" .Supplier.ICO }}"
        style: { size: 9, color: gray }

  - if: "not .Supplier.IsVATPayer"
    row: 4
    cols:
      - width: 12
        text: "{{ label \"pdf.not_vat_payer\" }}"
        style: { size: 8, italic: true, color: light_gray }

  - spacer: 8

  - row: 5
    cols:
      - width: 12
        text: "{{ label \"pdf.issued_for\" }}:"
        style: { size: 8, color: light_gray }

  - row: 5
    cols:
      - width: 12
        text: "{{ .Customer.Name }}"
        style: { size: 10, bold: true }

  - row: 4
    cols:
      - width: 12
        text: "{{ .Customer.Street }}, {{ .Customer.ZIP }} {{ .Customer.City }} | {{ labelf \"pdf.ico\" .Customer.ICO }}"
        style: { size: 9, color: gray }

  - spacer: 15

  # ── Payment info (with bank) ──
  - if: ".Options.HasBankInfo"
    line: { color: line_color, size_percent: 100 }

  - if: ".Options.HasBankInfo"
    row: 12
    cols:
      - width: 4
        texts:
          - text: "{{ label \"pdf.due_date\" }}"
            style: { size: 9, bold: true, top: 2 }
          - text: "{{ .Invoice.DueDate | date }}"
            style: { size: 9, bold: true, top: 6 }
      - width: 4
        text: "{{ label \"pdf.variable_symbol_short\" }}: {{ .Invoice.VariableSymbol }}"
        style: { size: 9, top: 2 }
      - width: 4
        text: "{{ label \"pdf.bank_account\" }} {{ .BankAccount.AccountNumber }}"
        style: { size: 9, align: right, top: 2 }

  - if: "and .Options.HasBankInfo .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 8
      - width: 4
        text: "{{ label \"pdf.taxable_date\" }}: {{ .Invoice.TaxableDate | date }}"
        style: { size: 9, align: right }

  - if: ".Options.HasBankInfo"
    line: { color: line_color, size_percent: 100 }

  # ── Payment info (without bank) ──
  - if: "not .Options.HasBankInfo"
    line: { color: line_color, size_percent: 100 }

  - if: "not .Options.HasBankInfo"
    row: 12
    cols:
      - width: 4
        texts:
          - text: "{{ label \"pdf.due_date\" }}"
            style: { size: 9, bold: true, top: 2 }
          - text: "{{ .Invoice.DueDate | date }}"
            style: { size: 9, bold: true, top: 6 }
      - width: 8

  - if: "and (not .Options.HasBankInfo) .Supplier.IsVATPayer"
    row: 6
    cols:
      - width: 8
      - width: 4
        text: "{{ label \"pdf.taxable_date\" }}: {{ .Invoice.TaxableDate | date }}"
        style: { size: 9, align: right }

  - if: "not .Options.HasBankInfo"
    line: { color: line_color, size_percent: 100 }

  - spacer: 15

  # ── Items ──
  - items:
      row_height: 7
      cols:
        - width: 8
          field: Description
          style: { size: 10 }
        - width: 2
          text: "{{ .Quantity | num0 }} x {{ .UnitPrice | num0 }}"
          style: { size: 9, align: right, color: gray }
        - width: 2
          field: Total
          format: money
          style: { size: 10, align: right }

  - spacer: 10

  # ── Totals ──
  - line: { color: divider, size_percent: 100 }

  - row: 12
    cols:
      - width: 8
      - width: 2
        text: "{{ label \"pdf.total\" }}"
        style: { size: 12, align: right, top: 3 }
      - width: 2
        text: "{{ .Invoice.Total | money }}"
        style: { size: 14, bold: true, align: right, top: 2 }



  # ── QR code ──
  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    spacer: 15

  - if: "and .Options.ShowQR (ne .Options.QRType \"none\")"
    row: 25
    cols:
      - width: 9
      - width: 3
        qr: true
`
