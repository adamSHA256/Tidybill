package backup

import (
	"time"

	"github.com/adamSHA256/tidybill/internal/model"
)

// FormatVersion is incremented only on breaking format changes.
const FormatVersion = 1

// ExportFile is the top-level JSON structure of a .tidybill file.
type ExportFile struct {
	Metadata      ExportMetadata      `json:"tidybill_export"`
	VatRates      []VatRateExport     `json:"vat_rates"`
	Settings      []SettingEntry      `json:"settings"`
	Suppliers     []model.Supplier    `json:"suppliers"`
	BankAccounts  []model.BankAccount `json:"bank_accounts"`
	Customers     []model.Customer    `json:"customers"`
	PDFTemplates  []PDFTemplateExport `json:"pdf_templates"`
	Items         []model.Item        `json:"items"`
	SmtpConfigs   []SmtpConfigExport  `json:"smtp_configs"`
	CustomerItems []CustomerItemExport `json:"customer_items"`
	Invoices      []InvoiceExport     `json:"invoices"`
	InvoiceItems  []model.InvoiceItem `json:"invoice_items"`
}

// ExportMetadata holds the header information for an export file.
type ExportMetadata struct {
	FormatVersion int            `json:"format_version"`
	AppVersion    string         `json:"app_version"`
	SchemaVersion int            `json:"schema_version"`
	ExportedAt    time.Time      `json:"exported_at"`
	DeviceID      string         `json:"device_id"`
	ExportMode    string         `json:"export_mode"`
	Filters       *ExportFilters `json:"filters"`
}

// ExportFilters controls what to include in a filtered export.
type ExportFilters struct {
	SupplierIDs            []string `json:"supplier_ids,omitempty"`
	SkipPaidOlderThanYears *int     `json:"skip_paid_older_than_years,omitempty"`
	DateFrom               string   `json:"date_from,omitempty"`
	DateTo                 string   `json:"date_to,omitempty"`
	ExcludeSettings        bool     `json:"exclude_settings,omitempty"`
}

// SettingEntry represents a single key-value setting for export.
type SettingEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SmtpConfigExport is SmtpConfig without password_encrypted.
type SmtpConfigExport struct {
	ID          string    `json:"id"`
	SupplierID  string    `json:"supplier_id"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	FromName    string    `json:"from_name"`
	FromEmail   string    `json:"from_email"`
	UseStartTLS bool      `json:"use_starttls"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// InvoiceExport is Invoice without pdf_path and without loaded relations / computed fields.
type InvoiceExport struct {
	ID             string              `json:"id"`
	InvoiceNumber  string              `json:"invoice_number"`
	SupplierID     string              `json:"supplier_id"`
	CustomerID     string              `json:"customer_id"`
	BankAccountID  string              `json:"bank_account_id"`
	Status         model.InvoiceStatus `json:"status"`
	IssueDate      time.Time           `json:"issue_date"`
	DueDate        time.Time           `json:"due_date"`
	PaidDate       *time.Time          `json:"paid_date"`
	TaxableDate    time.Time           `json:"taxable_date"`
	PaymentMethod  string              `json:"payment_method"`
	VariableSymbol string              `json:"variable_symbol"`
	Currency       string              `json:"currency"`
	ExchangeRate   float64             `json:"exchange_rate"`
	Subtotal       float64             `json:"subtotal"`
	VATTotal       float64             `json:"vat_total"`
	Total          float64             `json:"total"`
	Notes          string              `json:"notes"`
	InternalNotes  string              `json:"internal_notes"`
	EmailSentAt    *time.Time          `json:"email_sent_at"`
	Language       string              `json:"language"`
	TemplateID     string              `json:"template_id"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// PDFTemplateExport is PDFTemplate without preview_path.
type PDFTemplateExport struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TemplateCode string `json:"template_code"`
	ConfigJSON   string `json:"config_json"`
	IsDefault    bool   `json:"is_default"`
	SupplierID   string `json:"supplier_id"`
	Description  string `json:"description"`
	ShowLogo     bool   `json:"show_logo"`
	ShowQR       bool   `json:"show_qr"`
	ShowNotes    bool   `json:"show_notes"`
	SortOrder    int    `json:"sort_order"`
	IsBuiltin    bool   `json:"is_builtin"`
	YAMLSource   string `json:"yaml_source"`
	ParentID     string `json:"parent_id"`
}

// CustomerItemExport is CustomerItem without joined fields.
type CustomerItemExport struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	ItemID       string    `json:"item_id"`
	LastPrice    float64   `json:"last_price"`
	LastQuantity float64   `json:"last_quantity"`
	UsageCount   int       `json:"usage_count"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

// VatRateExport represents a vat_rates row for export.
type VatRateExport struct {
	ID        string  `json:"id"`
	Rate      float64 `json:"rate"`
	Name      string  `json:"name"`
	IsDefault bool    `json:"is_default"`
	Country   string  `json:"country"`
}

// Import mode constants.
const (
	ImportModeFullReplace = "full_replace"
	ImportModeSmartMerge  = "smart_merge"
	ImportModeForce       = "force"
	ImportModePreview     = "preview"
)

// ImportOptions configures import behavior.
type ImportOptions struct {
	Mode                  string `json:"mode"`
	InvoiceNumberConflict string `json:"invoice_number_conflict"` // "skip", "auto_suffix"
	Passphrase            string `json:"passphrase,omitempty"`
}

// ImportReport is returned after import completes (or after preview).
type ImportReport struct {
	Mode       string                  `json:"mode"`
	StartedAt  time.Time               `json:"started_at"`
	FinishedAt time.Time               `json:"finished_at"`
	Summary    ImportSummary           `json:"summary"`
	Details    map[string]TableSummary `json:"details"`
	Conflicts  []ImportConflict        `json:"conflicts"`
	Warnings   []ImportWarning         `json:"warnings"`
}

// ImportSummary holds aggregate counts for an import operation.
type ImportSummary struct {
	ToInsert  int `json:"to_insert"`
	ToUpdate  int `json:"to_update"`
	ToSkip    int `json:"to_skip"`
	Conflicts int `json:"conflicts"`
	Warnings  int `json:"warnings"`
}

// TableSummary holds per-table counts for an import operation.
type TableSummary struct {
	Insert    int `json:"insert"`
	Update    int `json:"update"`
	Skip      int `json:"skip"`
	Conflicts int `json:"conflicts"`
}

// ImportConflict describes a conflict encountered during import.
type ImportConflict struct {
	Table       string `json:"table"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
}

// ImportWarning describes a non-fatal issue encountered during import.
type ImportWarning struct {
	Table       string `json:"table"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
}
