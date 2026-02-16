package service

// TemplateEditingAIPrompt is shown to users so they can paste it into an AI assistant
// when editing a YAML template. It explains the format, available variables, and helpers.
const TemplateEditingAIPrompt = `You are helping edit an invoice PDF template written in YAML format.
The template uses a declarative layout that gets rendered to PDF via the Maroto library.

## Structure

The YAML has these top-level keys:
- name: Template display name
- margins: { left, top, right } in mm
- colors: Named color definitions as { r, g, b } (0-255)
- layout: Array of layout elements

## Layout Elements

Each element in the layout array can be one of:

### Row with columns (single text per column)
- row: <height>       # Row height in mm
  cols:
    - width: <1-12>   # Column width (12-column grid)
      text: "..."      # Text content (supports {{ }} template expressions)
      style: { size: 10, bold: true, align: right, color: <color_name>, top: 0, left: 0 }
      cell: { border: full, bg_color: <color_name>, border_color: <color_name> }  # Optional cell styling

### Row with multi-text columns (multiple positioned text items in one column)
- row: 80
  cols:
    - width: 4
      texts:
        - text: "SUPPLIER"
          style: { size: 10, bold: true }
        - text: "{{ .Supplier.Name }}"
          style: { size: 10, bold: true, top: 5 }
        - text: "{{ .Supplier.Street }}"
          style: { size: 9, top: 10 }
      cell: { border: full }  # Optional: applies to the entire column

Use "texts:" (array) when you need multiple text items with different styles/positions
within a single column. Each text item has its own "top:" offset for vertical positioning.

### Spacer
- spacer: <height>    # Empty space in mm

### Line
- line: { color: <color_name> }   # Horizontal divider line

### Items table (loops over invoice items)
- items:
    row_height: 7
    header_cell: { border: full, bg_color: <color_name> }  # Applied to all header columns
    row_cell: { border: full, border_color: <color_name> }  # Applied to all data columns
    header:            # Optional header row
      - width: 5
        text: "Description"
        style: { ... }
    cols:              # Repeated for each item
      - width: 5
        field: Description    # Direct field access: Description, Quantity, Unit, UnitPrice, VATRate, VATAmount, Subtotal, Total
        format: money         # Optional: "money" formats as currency
        style: { ... }
      - width: 2
        text: "{{ .Quantity | num0 }} {{ .Unit }}"   # Template expression with item fields
        style: { ... }

### Notes section
- notes: true          # Renders invoice notes if ShowNotes is enabled

### Special columns
- qr: true             # QR payment code
- logo: true           # Supplier logo image

### Conditional rendering
- if: "<condition>"    # Go template condition, element only renders if true
  row: 10
  cols: [...]

Common conditions:
  "and .Options.ShowQR (ne .Options.QRType \"none\")"
  "not .Supplier.IsVATPayer"
  "gt .Invoice.VATTotal 0"
  "and .Options.ShowLogo .Supplier.LogoPath"

## Cell Styling

Columns and items can have cell-level styling:
  cell: { border: full, bg_color: <color>, border_color: <color> }

Border values: full, left, right, top, bottom
Colors reference named colors from the top-level "colors:" section.

For items tables, use header_cell and row_cell to style all cells uniformly:
  header_cell: { border: full, bg_color: header_bg }
  row_cell: { border: full, border_color: light_border }

## Template Variables

Invoice:
  .Invoice.InvoiceNumber, .Invoice.IssueDate, .Invoice.DueDate, .Invoice.TaxableDate
  .Invoice.PaymentMethod, .Invoice.VariableSymbol, .Invoice.Currency
  .Invoice.Subtotal, .Invoice.VATTotal, .Invoice.Total
  .Invoice.Notes, .Invoice.Status, .Invoice.Language

Supplier:
  .Supplier.Name, .Supplier.Street, .Supplier.City, .Supplier.ZIP, .Supplier.Country
  .Supplier.ICO, .Supplier.DIC, .Supplier.Phone, .Supplier.Email, .Supplier.Website
  .Supplier.LogoPath, .Supplier.IsVATPayer, .Supplier.InvoicePrefix

Customer:
  .Customer.Name, .Customer.Street, .Customer.City, .Customer.ZIP
  .Customer.Region, .Customer.Country, .Customer.ICO, .Customer.DIC
  .Customer.Email, .Customer.Phone

BankAccount:
  .BankAccount.Name, .BankAccount.AccountNumber, .BankAccount.IBAN
  .BankAccount.SWIFT, .BankAccount.Currency

Options:
  .Options.ShowLogo, .Options.ShowQR, .Options.ShowNotes, .Options.QRType

## Helper Functions (use with | pipe syntax)
  | date    - Format time as "02.01.2006"
  | money   - Format number as "79 600,00 CZK"
  | num0    - Format number with 0 decimals: "250"
  label "key"  - Get translated label, e.g. {{ label "pdf.total" }}
  labelf "key" arg1 arg2  - Formatted label (for keys with %s), e.g. {{ labelf "pdf.ico" .Supplier.ICO }}

## Style Properties
  size: Font size in points (default 10)
  bold: true/false
  italic: true/false
  align: left (default), right, center
  color: Reference to a named color from the colors section
  top: Vertical offset in mm (for positioning within a multi-text column)
  left: Left padding in mm
  right: Right padding in mm

## Column Grid
  Columns use a 12-unit grid (like Bootstrap). Width values must sum to 12 per row.
  Empty columns (no text/field) act as spacers.

## Tips
- Keep row heights proportional - typical text rows are 5-8mm, headers 10-20mm
- Use spacer elements between sections for visual separation
- Colors are defined at the top and referenced by name in styles
- The "items" section automatically loops - you define one row template
- Use "texts:" with "top:" offsets for complex layouts (e.g. supplier/customer blocks)
- Use "labelf" for i18n keys that contain %s placeholders (e.g. "pdf.ico" = "IČO: %s")
- Use "label" for i18n keys without placeholders (e.g. "pdf.total" = "CELKEM:")
- Test changes by generating a preview after each edit
`
