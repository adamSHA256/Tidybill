package service

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"gopkg.in/yaml.v2"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

// DeclarativeTemplate renders PDF from a YAML layout definition.
type DeclarativeTemplate struct {
	Source string
	parsed *TemplateLayout
}

// TemplateLayout is the top-level YAML structure.
type TemplateLayout struct {
	Name    string               `yaml:"name"`
	Margins *YAMLMargins         `yaml:"margins"`
	Colors  map[string]YAMLColor `yaml:"colors"`
	Layout  []YAMLElement        `yaml:"layout"`
}

type YAMLMargins struct {
	Left  float64 `yaml:"left"`
	Top   float64 `yaml:"top"`
	Right float64 `yaml:"right"`
}

type YAMLColor struct {
	R int `yaml:"r"`
	G int `yaml:"g"`
	B int `yaml:"b"`
}

type YAMLStyle struct {
	Size   float64 `yaml:"size"`
	Bold   bool    `yaml:"bold"`
	Italic bool    `yaml:"italic"`
	Align  string  `yaml:"align"`
	Top    float64 `yaml:"top"`
	Left   float64 `yaml:"left"`
	Right  float64 `yaml:"right"`
	Color  string  `yaml:"color"`
}

// YAMLText represents a single text item within a multi-text column.
type YAMLText struct {
	Text  string    `yaml:"text"`
	Style YAMLStyle `yaml:"style"`
}

// YAMLCell defines cell-level styling (borders, background).
type YAMLCell struct {
	Border      string `yaml:"border"`
	BgColor     string `yaml:"bg_color"`
	BorderColor string `yaml:"border_color"`
}

type YAMLCol struct {
	Width  int        `yaml:"width"`
	Text   string     `yaml:"text"`
	Texts  []YAMLText `yaml:"texts"`
	Field  string     `yaml:"field"`
	Format string     `yaml:"format"`
	Style  YAMLStyle  `yaml:"style"`
	Cell   *YAMLCell  `yaml:"cell"`
	QR     bool       `yaml:"qr"`
	Logo   bool       `yaml:"logo"`
}

type YAMLItems struct {
	RowHeight  float64   `yaml:"row_height"`
	Cols       []YAMLCol `yaml:"cols"`
	Header     []YAMLCol `yaml:"header"`
	HeaderCell *YAMLCell `yaml:"header_cell"`
	RowCell    *YAMLCell `yaml:"row_cell"`
}

type YAMLLineStyle struct {
	Color       string  `yaml:"color"`
	SizePercent float64 `yaml:"size_percent"`
}

type YAMLElement struct {
	Row    interface{}    `yaml:"row"`
	Cols   []YAMLCol      `yaml:"cols"`
	Spacer float64        `yaml:"spacer"`
	Line   *YAMLLineStyle `yaml:"line"`
	Items  *YAMLItems     `yaml:"items"`
	Notes  bool           `yaml:"notes"`
	If     string         `yaml:"if"`
}

// templateData wraps all data passed to Go template expressions.
type templateData struct {
	Invoice     interface{}
	Supplier    interface{}
	Customer    interface{}
	BankAccount interface{}
	Items       interface{}
	Options     *TemplateOptions
	Lang        i18n.Lang
}

func NewDeclarativeRenderer(yamlSource string) (*DeclarativeTemplate, error) {
	d := &DeclarativeTemplate{Source: yamlSource}
	if err := d.parse(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DeclarativeTemplate) parse() error {
	d.parsed = &TemplateLayout{}
	return yaml.Unmarshal([]byte(d.Source), d.parsed)
}

func (d *DeclarativeTemplate) Margins() TemplateMargins {
	if d.parsed != nil && d.parsed.Margins != nil {
		return TemplateMargins{
			Left:  d.parsed.Margins.Left,
			Top:   d.parsed.Margins.Top,
			Right: d.parsed.Margins.Right,
		}
	}
	return TemplateMargins{Left: 10, Top: 10, Right: 10}
}

func (d *DeclarativeTemplate) Render(m core.Maroto, data *InvoiceData, opts *TemplateOptions) {
	if d.parsed == nil {
		return
	}

	td := &templateData{
		Invoice:     data.Invoice,
		Supplier:    data.Supplier,
		Customer:    data.Customer,
		BankAccount: data.BankAccount,
		Items:       data.Items,
		Options:     opts,
		Lang:        invoiceLang(data),
	}

	for _, elem := range d.parsed.Layout {
		d.renderElement(m, elem, data, opts, td)
	}
}

func (d *DeclarativeTemplate) renderElement(m core.Maroto, elem YAMLElement, data *InvoiceData, opts *TemplateOptions, td *templateData) {
	if elem.If != "" {
		if !d.evalCondition(elem.If, td) {
			return
		}
	}

	if elem.Spacer > 0 {
		m.AddRow(elem.Spacer)
		return
	}

	if elem.Line != nil {
		lineProp := props.Line{Color: d.resolveColor(elem.Line.Color)}
		if elem.Line.SizePercent > 0 {
			lineProp.SizePercent = elem.Line.SizePercent
		}
		m.AddRows(row.New(1).Add(
			line.NewCol(12, lineProp),
		))
		return
	}

	if elem.Items != nil {
		d.renderItems(m, elem.Items, data, opts, td)
		return
	}

	if elem.Notes {
		if opts.ShowNotes && data.Invoice.Notes != "" {
			m.AddRows(row.New(15).Add(
				text.NewCol(12, data.Invoice.Notes, props.Text{Size: 9}),
			))
		}
		return
	}

	if elem.Cols != nil {
		height := d.rowHeight(elem.Row)
		if height <= 0 {
			height = 8
		}
		cols := d.buildCols(elem.Cols, data, opts, td)
		m.AddRows(row.New(height).Add(cols...))
		return
	}

	if elem.Row != nil && elem.Cols == nil {
		height := d.rowHeight(elem.Row)
		if height > 0 {
			m.AddRow(height)
		}
	}
}

func (d *DeclarativeTemplate) buildCols(yamlCols []YAMLCol, data *InvoiceData, opts *TemplateOptions, td *templateData) []core.Col {
	var cols []core.Col
	for _, yc := range yamlCols {
		width := yc.Width
		if width <= 0 {
			width = 1
		}

		// QR code column
		if yc.QR {
			spayd := GenerateQRPayload(opts.QRType, data)
			if spayd != "" {
				cols = append(cols, code.NewQrCol(width, spayd, props.Rect{Percent: 100}))
			} else {
				cols = append(cols, col.New(width))
			}
			continue
		}

		// Logo column
		if yc.Logo {
			if opts.ShowLogo && data.Supplier.LogoPath != "" {
				cols = append(cols, image.NewFromFileCol(width, data.Supplier.LogoPath, props.Rect{
					Percent: 80,
					Center:  true,
				}))
			} else {
				cols = append(cols, col.New(width))
			}
			continue
		}

		// Multi-text column (multiple text items in one column)
		if len(yc.Texts) > 0 {
			c := col.New(width)
			var components []core.Component
			for _, yt := range yc.Texts {
				content := d.evalText(yt.Text, td)
				textProps := d.buildTextProps(yt.Style)
				components = append(components, text.New(content, textProps))
			}
			c = c.Add(components...)
			if yc.Cell != nil {
				c = c.WithStyle(d.buildCellStyle(yc.Cell))
			}
			cols = append(cols, c)
			continue
		}

		// Empty column (no text)
		if yc.Text == "" && yc.Field == "" {
			c := col.New(width)
			if yc.Cell != nil {
				c = c.WithStyle(d.buildCellStyle(yc.Cell))
			}
			cols = append(cols, c)
			continue
		}

		// Single text column
		content := d.evalText(yc.Text, td)
		textProps := d.buildTextProps(yc.Style)
		if yc.Cell != nil {
			c := col.New(width).Add(text.New(content, textProps))
			c = c.WithStyle(d.buildCellStyle(yc.Cell))
			cols = append(cols, c)
		} else {
			cols = append(cols, text.NewCol(width, content, textProps))
		}
	}
	return cols
}

func (d *DeclarativeTemplate) renderItems(m core.Maroto, items *YAMLItems, data *InvoiceData, opts *TemplateOptions, td *templateData) {
	rowHeight := items.RowHeight
	if rowHeight <= 0 {
		rowHeight = 7
	}

	// Render header row
	if items.Header != nil {
		var headerCols []core.Col
		for _, yc := range items.Header {
			width := yc.Width
			if width <= 0 {
				width = 1
			}
			content := d.evalText(yc.Text, td)
			textProps := d.buildTextProps(yc.Style)
			c := col.New(width).Add(text.New(content, textProps))
			if items.HeaderCell != nil {
				c = c.WithStyle(d.buildCellStyle(items.HeaderCell))
			} else if yc.Cell != nil {
				c = c.WithStyle(d.buildCellStyle(yc.Cell))
			}
			headerCols = append(headerCols, c)
		}
		m.AddRows(row.New(rowHeight).Add(headerCols...))
	}

	currency := data.Invoice.Currency

	for _, item := range data.Items {
		var cols []core.Col
		for _, yc := range items.Cols {
			width := yc.Width
			if width <= 0 {
				width = 1
			}

			var content string
			if yc.Field != "" {
				content = d.resolveItemField(yc.Field, item, currency, yc.Format)
			} else if yc.Text != "" {
				content = d.evalItemText(yc.Text, item, td)
			}

			if content == "" && yc.Field == "" && yc.Text == "" {
				c := col.New(width)
				if items.RowCell != nil {
					c = c.WithStyle(d.buildCellStyle(items.RowCell))
				}
				cols = append(cols, c)
				continue
			}

			textProps := d.buildTextProps(yc.Style)
			c := col.New(width).Add(text.New(content, textProps))
			if items.RowCell != nil {
				c = c.WithStyle(d.buildCellStyle(items.RowCell))
			} else if yc.Cell != nil {
				c = c.WithStyle(d.buildCellStyle(yc.Cell))
			}
			cols = append(cols, c)
		}
		m.AddRows(row.New(rowHeight).Add(cols...))
	}
}

func (d *DeclarativeTemplate) resolveItemField(field string, item interface{}, currency, format string) string {
	switch strings.ToLower(field) {
	case "description":
		return getItemString(item, "Description")
	case "quantity":
		v := getItemFloat(item, "Quantity")
		return fmt.Sprintf("%.0f", v)
	case "unit":
		return getItemString(item, "Unit")
	case "unitprice", "unit_price":
		v := getItemFloat(item, "UnitPrice")
		if format == "money" {
			return formatMoneyDefault(v, currency)
		}
		return fmt.Sprintf("%.2f", v)
	case "vatrate", "vat_rate":
		v := getItemFloat(item, "VATRate")
		return fmt.Sprintf("%.0f", v)
	case "vatamount", "vat_amount":
		v := getItemFloat(item, "VATAmount")
		if format == "money" {
			return formatMoneyDefault(v, currency)
		}
		return fmt.Sprintf("%.2f", v)
	case "subtotal":
		v := getItemFloat(item, "Subtotal")
		if format == "money" {
			return formatMoneyDefault(v, currency)
		}
		return fmt.Sprintf("%.2f", v)
	case "total":
		v := getItemFloat(item, "Total")
		if format == "money" {
			return formatMoneyDefault(v, currency)
		}
		return formatSimple(v, currency)
	default:
		return ""
	}
}

func (d *DeclarativeTemplate) buildTextProps(style YAMLStyle) props.Text {
	p := props.Text{
		Size:  style.Size,
		Top:   style.Top,
		Left:  style.Left,
		Right: style.Right,
	}
	if p.Size <= 0 {
		p.Size = 10
	}
	if style.Bold && style.Italic {
		p.Style = fontstyle.BoldItalic
	} else if style.Bold {
		p.Style = fontstyle.Bold
	} else if style.Italic {
		p.Style = fontstyle.Italic
	}
	switch strings.ToLower(style.Align) {
	case "right":
		p.Align = align.Right
	case "center":
		p.Align = align.Center
	}
	if style.Color != "" {
		p.Color = d.resolveColor(style.Color)
	}
	return p
}

func (d *DeclarativeTemplate) buildCellStyle(cell *YAMLCell) *props.Cell {
	if cell == nil {
		return nil
	}
	cs := &props.Cell{}
	switch strings.ToLower(cell.Border) {
	case "full":
		cs.BorderType = border.Full
	case "left":
		cs.BorderType = border.Left
	case "right":
		cs.BorderType = border.Right
	case "top":
		cs.BorderType = border.Top
	case "bottom":
		cs.BorderType = border.Bottom
	}
	if cell.BgColor != "" {
		cs.BackgroundColor = d.resolveColor(cell.BgColor)
	}
	if cell.BorderColor != "" {
		cs.BorderColor = d.resolveColor(cell.BorderColor)
	}
	return cs
}

func (d *DeclarativeTemplate) resolveColor(name string) *props.Color {
	if d.parsed == nil || name == "" {
		return nil
	}
	if c, ok := d.parsed.Colors[name]; ok {
		return &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	return nil
}

func (d *DeclarativeTemplate) evalText(tmplStr string, td *templateData) string {
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr
	}
	t, err := template.New("").Funcs(templateFuncs(td.Lang)).Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, td); err != nil {
		return tmplStr
	}
	return buf.String()
}

func (d *DeclarativeTemplate) evalItemText(tmplStr string, item interface{}, td *templateData) string {
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr
	}
	t, err := template.New("").Funcs(templateFuncs(td.Lang)).Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, item); err != nil {
		return tmplStr
	}
	return buf.String()
}

func (d *DeclarativeTemplate) evalCondition(cond string, td *templateData) bool {
	tmplStr := fmt.Sprintf("{{if %s}}true{{end}}", cond)
	t, err := template.New("").Funcs(templateFuncs(td.Lang)).Parse(tmplStr)
	if err != nil {
		return false
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, td); err != nil {
		return false
	}
	return buf.String() == "true"
}

func (d *DeclarativeTemplate) rowHeight(v interface{}) float64 {
	switch h := v.(type) {
	case float64:
		return h
	case int:
		return float64(h)
	default:
		return 0
	}
}

// templateFuncs returns the Go template functions available in YAML templates.
func templateFuncs(lang i18n.Lang) template.FuncMap {
	return template.FuncMap{
		"date": func(t interface{}) string {
			if tm, ok := t.(interface{ Format(string) string }); ok {
				return tm.Format("02.01.2006")
			}
			return fmt.Sprintf("%v", t)
		},
		"money": func(args ...interface{}) string {
			if len(args) == 0 {
				return ""
			}
			amount, ok := args[0].(float64)
			if !ok {
				return fmt.Sprintf("%v", args[0])
			}
			currency := ""
			if len(args) > 1 {
				currency, _ = args[1].(string)
			}
			return formatMoneyDefault(amount, currency)
		},
		"num0": func(v interface{}) string {
			if f, ok := v.(float64); ok {
				return fmt.Sprintf("%.0f", f)
			}
			return fmt.Sprintf("%v", v)
		},
		"label": func(key string) string {
			return i18n.TForLang(lang, key)
		},
		"labelf": func(key string, args ...interface{}) string {
			return i18n.TfForLang(lang, key, args...)
		},
		"invoiceTitle": func(number string, isVATPayer bool) string {
			return invoiceTitle(lang, number, isVATPayer)
		},
	}
}

// Helper functions to extract fields from InvoiceItem via reflection-free approach
func getItemString(item interface{}, field string) string {
	if m, ok := item.(map[string]interface{}); ok {
		if v, ok := m[field]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return fmt.Sprintf("%v", getStructField(item, field))
}

func getItemFloat(item interface{}, field string) float64 {
	v := getStructField(item, field)
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func getStructField(item interface{}, field string) interface{} {
	tmpl := fmt.Sprintf("{{.%s}}", field)
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, item); err != nil {
		return nil
	}

	s := buf.String()
	switch field {
	case "Quantity", "UnitPrice", "VATRate", "Subtotal", "VATAmount", "Total":
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}
	return s
}

// ValidateYAML checks if a YAML template is valid and can be parsed.
func ValidateYAML(yamlSource string) error {
	layout := &TemplateLayout{}
	if err := yaml.Unmarshal([]byte(yamlSource), layout); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}
	if layout.Layout == nil || len(layout.Layout) == 0 {
		return fmt.Errorf("template must have at least one layout element")
	}
	return nil
}
