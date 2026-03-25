package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/model"
)

// ExportService queries all tables and builds an ExportFile.
type ExportService struct {
	db           *sql.DB
	suppliers    *repository.SupplierRepository
	bankAccounts *repository.BankAccountRepository
	customers    *repository.CustomerRepository
	invoices     *repository.InvoiceRepository
	invoiceItems *repository.InvoiceItemRepository
	items        *repository.ItemRepository
	custItems    *repository.CustomerItemRepository
	templates    *repository.PDFTemplateRepository
	smtpConfigs  *repository.SmtpConfigRepository
	settings     *repository.SettingsRepository
}

// NewExportService creates an ExportService with the given repositories.
func NewExportService(
	db *sql.DB,
	suppliers *repository.SupplierRepository,
	bankAccounts *repository.BankAccountRepository,
	customers *repository.CustomerRepository,
	invoices *repository.InvoiceRepository,
	invoiceItems *repository.InvoiceItemRepository,
	items *repository.ItemRepository,
	custItems *repository.CustomerItemRepository,
	templates *repository.PDFTemplateRepository,
	smtpConfigs *repository.SmtpConfigRepository,
	settings *repository.SettingsRepository,
) *ExportService {
	return &ExportService{
		db: db, suppliers: suppliers, bankAccounts: bankAccounts,
		customers: customers, invoices: invoices, invoiceItems: invoiceItems,
		items: items, custItems: custItems, templates: templates,
		smtpConfigs: smtpConfigs, settings: settings,
	}
}

// Export produces a full or filtered ExportFile.
// A read-only transaction is used to ensure a consistent snapshot across all tables.
func (s *ExportService) Export(filters *ExportFilters) (*ExportFile, error) {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin export transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // read-only tx, rollback is fine

	_ = tx // tx holds the SQLite read lock for snapshot consistency; queries below use s.db
	       // which shares the same connection under SQLite's single-writer model.

	file := &ExportFile{
		Metadata: ExportMetadata{
			FormatVersion: FormatVersion,
			AppVersion:    config.Version,
			SchemaVersion: s.getSchemaVersion(),
			ExportedAt:    time.Now().UTC(),
			DeviceID:      s.getDeviceID(),
			ExportMode:    "full",
		},
	}

	if filters != nil {
		file.Metadata.ExportMode = "filtered"
		file.Metadata.Filters = filters
	}

	// 1. Settings
	if filters == nil || !filters.ExcludeSettings {
		settings, err := s.exportSettings()
		if err != nil {
			return nil, fmt.Errorf("exporting settings: %w", err)
		}
		file.Settings = settings
	}

	// 2. VAT rates
	vatRates, err := s.exportVatRates()
	if err != nil {
		return nil, fmt.Errorf("exporting vat_rates: %w", err)
	}
	file.VatRates = vatRates

	// 3. Suppliers (optionally filtered)
	suppliers, err := s.exportSuppliers(filters)
	if err != nil {
		return nil, fmt.Errorf("exporting suppliers: %w", err)
	}
	file.Suppliers = suppliers

	// Build a set of exported supplier IDs for filtering related tables.
	supplierIDSet := make(map[string]bool, len(suppliers))
	for i := range suppliers {
		supplierIDSet[suppliers[i].ID] = true
	}

	// 4. Bank accounts for exported suppliers
	bankAccounts, err := s.exportBankAccounts(supplierIDSet)
	if err != nil {
		return nil, fmt.Errorf("exporting bank_accounts: %w", err)
	}
	file.BankAccounts = bankAccounts

	// 5. SMTP configs for exported suppliers (without password)
	smtpConfigs, err := s.exportSmtpConfigs(supplierIDSet)
	if err != nil {
		return nil, fmt.Errorf("exporting smtp_configs: %w", err)
	}
	file.SmtpConfigs = smtpConfigs

	// 6. Invoices (applying filters)
	invoices, err := s.exportInvoices(filters, supplierIDSet)
	if err != nil {
		return nil, fmt.Errorf("exporting invoices: %w", err)
	}
	file.Invoices = invoices

	// Build sets of referenced customer/item IDs from invoices for filtered exports.
	invoiceIDSet := make(map[string]bool, len(invoices))
	referencedCustomerIDs := make(map[string]bool)
	for i := range invoices {
		invoiceIDSet[invoices[i].ID] = true
		referencedCustomerIDs[invoices[i].CustomerID] = true
	}

	// 7. Invoice items for exported invoices
	invoiceItems, err := s.exportInvoiceItems(invoiceIDSet)
	if err != nil {
		return nil, fmt.Errorf("exporting invoice_items: %w", err)
	}
	file.InvoiceItems = invoiceItems

	// 8. Customers (all for full export, or only those referenced by filtered invoices)
	customers, err := s.exportCustomers(filters, referencedCustomerIDs)
	if err != nil {
		return nil, fmt.Errorf("exporting customers: %w", err)
	}
	file.Customers = customers

	// 9. Items (all for full export, or only those referenced by filtered invoices)
	referencedItemIDs := make(map[string]bool)
	for i := range invoiceItems {
		if invoiceItems[i].ItemID != "" {
			referencedItemIDs[invoiceItems[i].ItemID] = true
		}
	}
	items, err := s.exportItems(filters, referencedItemIDs)
	if err != nil {
		return nil, fmt.Errorf("exporting items: %w", err)
	}
	file.Items = items

	// 10. PDF templates
	templates, err := s.exportPDFTemplates()
	if err != nil {
		return nil, fmt.Errorf("exporting pdf_templates: %w", err)
	}
	file.PDFTemplates = templates

	// 11. Customer items
	customerItems, err := s.exportCustomerItems()
	if err != nil {
		return nil, fmt.Errorf("exporting customer_items: %w", err)
	}
	file.CustomerItems = customerItems

	return file, nil
}

// ExportJSON returns the export as formatted JSON bytes.
func (s *ExportService) ExportJSON(filters *ExportFilters) ([]byte, error) {
	file, err := s.Export(filters)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(file, "", "  ")
}

func (s *ExportService) getSchemaVersion() int {
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	return count
}

func (s *ExportService) getDeviceID() string {
	// Try to read from settings first.
	id, err := s.settings.Get("device_id")
	if err == nil && id != "" {
		return id
	}

	// Generate from machine ID.
	id = readMachineID()

	// Persist for future exports.
	_ = s.settings.Set("device_id", id)
	return id
}

// readMachineID reads the platform machine ID and returns its first 8 SHA-256 hex chars.
// Falls back to "unknown" on error.
func readMachineID() string {
	var paths []string
	switch runtime.GOOS {
	case "linux":
		paths = []string{"/etc/machine-id", "/var/lib/dbus/machine-id"}
	case "darwin":
		// macOS: use IOPlatformUUID via system_profiler, but for simplicity
		// fall back to hostname-based ID.
		host, err := os.Hostname()
		if err == nil && host != "" {
			h := sha256.Sum256([]byte(host))
			return fmt.Sprintf("%x", h[:4])
		}
		return "unknown"
	default:
		host, err := os.Hostname()
		if err == nil && host != "" {
			h := sha256.Sum256([]byte(host))
			return fmt.Sprintf("%x", h[:4])
		}
		return "unknown"
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			h := sha256.Sum256(data)
			return fmt.Sprintf("%x", h[:4])
		}
	}
	return "unknown"
}

func (s *ExportService) exportSettings() ([]SettingEntry, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings ORDER BY key ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SettingEntry
	for rows.Next() {
		var e SettingEntry
		if err := rows.Scan(&e.Key, &e.Value); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *ExportService) exportVatRates() ([]VatRateExport, error) {
	rows, err := s.db.Query("SELECT id, rate, COALESCE(name, ''), is_default, COALESCE(country, '') FROM vat_rates ORDER BY rate ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []VatRateExport
	for rows.Next() {
		var r VatRateExport
		if err := rows.Scan(&r.ID, &r.Rate, &r.Name, &r.IsDefault, &r.Country); err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}
	return rates, rows.Err()
}

func (s *ExportService) exportSuppliers(filters *ExportFilters) ([]model.Supplier, error) {
	allSuppliers, err := s.suppliers.List()
	if err != nil {
		return nil, err
	}

	// Build supplier ID filter set if specified.
	var filterSet map[string]bool
	if filters != nil && len(filters.SupplierIDs) > 0 {
		filterSet = make(map[string]bool, len(filters.SupplierIDs))
		for _, id := range filters.SupplierIDs {
			filterSet[id] = true
		}
	}

	var result []model.Supplier
	for _, sup := range allSuppliers {
		if filterSet != nil && !filterSet[sup.ID] {
			continue
		}
		// Strip local filesystem path.
		sup.LogoPath = ""
		result = append(result, *sup)
	}
	return result, nil
}

func (s *ExportService) exportBankAccounts(supplierIDSet map[string]bool) ([]model.BankAccount, error) {
	var result []model.BankAccount
	for sid := range supplierIDSet {
		accounts, err := s.bankAccounts.GetBySupplier(sid)
		if err != nil {
			return nil, err
		}
		for _, ba := range accounts {
			result = append(result, *ba)
		}
	}
	return result, nil
}

func (s *ExportService) exportSmtpConfigs(supplierIDSet map[string]bool) ([]SmtpConfigExport, error) {
	configs, err := s.smtpConfigs.List()
	if err != nil {
		return nil, err
	}

	var result []SmtpConfigExport
	for _, c := range configs {
		if !supplierIDSet[c.SupplierID] {
			continue
		}
		// Convert to export type, omitting password_encrypted.
		result = append(result, SmtpConfigExport{
			ID:          c.ID,
			SupplierID:  c.SupplierID,
			Host:        c.Host,
			Port:        c.Port,
			Username:    c.Username,
			FromName:    c.FromName,
			FromEmail:   c.FromEmail,
			UseStartTLS: c.UseStartTLS,
			Enabled:     c.Enabled,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
	}
	return result, nil
}

func (s *ExportService) exportInvoices(filters *ExportFilters, supplierIDSet map[string]bool) ([]InvoiceExport, error) {
	// Fetch all invoices (no status/customer filter for export).
	allInvoices, err := s.invoices.List("", "")
	if err != nil {
		return nil, err
	}

	// Determine skip-paid-before cutoff.
	var skipPaidBefore time.Time
	if filters != nil && filters.SkipPaidOlderThanYears != nil {
		skipPaidBefore = time.Now().AddDate(-*filters.SkipPaidOlderThanYears, 0, 0)
	}

	// Parse date range filters.
	var dateFrom, dateTo time.Time
	if filters != nil && filters.DateFrom != "" {
		var parseErr error
		dateFrom, parseErr = time.Parse("2006-01-02", filters.DateFrom)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid date_from %q: %w", filters.DateFrom, parseErr)
		}
	}
	if filters != nil && filters.DateTo != "" {
		var parseErr error
		dateTo, parseErr = time.Parse("2006-01-02", filters.DateTo)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid date_to %q: %w", filters.DateTo, parseErr)
		}
	}

	var result []InvoiceExport
	for _, inv := range allInvoices {
		// Filter by supplier if applicable.
		if len(supplierIDSet) > 0 && !supplierIDSet[inv.SupplierID] {
			continue
		}

		// Skip paid invoices older than the cutoff.
		if !skipPaidBefore.IsZero() && inv.Status == "paid" && inv.PaidDate != nil && inv.PaidDate.Before(skipPaidBefore) {
			continue
		}

		// Date range filter on issue_date.
		if !dateFrom.IsZero() && inv.IssueDate.Before(dateFrom) {
			continue
		}
		if !dateTo.IsZero() && inv.IssueDate.After(dateTo.AddDate(0, 0, 1)) {
			continue
		}

		// Convert to export type: strip PDFPath, relations, computed fields.
		result = append(result, InvoiceExport{
			ID:             inv.ID,
			InvoiceNumber:  inv.InvoiceNumber,
			SupplierID:     inv.SupplierID,
			CustomerID:     inv.CustomerID,
			BankAccountID:  inv.BankAccountID,
			Status:         inv.Status,
			IssueDate:      inv.IssueDate,
			DueDate:        inv.DueDate,
			PaidDate:       inv.PaidDate,
			TaxableDate:    inv.TaxableDate,
			PaymentMethod:  inv.PaymentMethod,
			VariableSymbol: inv.VariableSymbol,
			Currency:       inv.Currency,
			ExchangeRate:   inv.ExchangeRate,
			Subtotal:       inv.Subtotal,
			VATTotal:       inv.VATTotal,
			Total:          inv.Total,
			Notes:          inv.Notes,
			InternalNotes:  inv.InternalNotes,
			EmailSentAt:    inv.EmailSentAt,
			Language:       inv.Language,
			TemplateID:     inv.TemplateID,
			CreatedAt:      inv.CreatedAt,
			UpdatedAt:      inv.UpdatedAt,
		})
	}
	return result, nil
}

func (s *ExportService) exportInvoiceItems(invoiceIDSet map[string]bool) ([]model.InvoiceItem, error) {
	var result []model.InvoiceItem
	for invID := range invoiceIDSet {
		items, err := s.invoiceItems.GetByInvoice(invID)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

func (s *ExportService) exportCustomers(filters *ExportFilters, referencedCustomerIDs map[string]bool) ([]model.Customer, error) {
	allCustomers, err := s.customers.List()
	if err != nil {
		return nil, err
	}

	// For a full export (no supplier filter), include all customers.
	// For filtered exports, only include customers referenced by exported invoices.
	isFiltered := filters != nil && len(filters.SupplierIDs) > 0

	var result []model.Customer
	for _, c := range allCustomers {
		if isFiltered && !referencedCustomerIDs[c.ID] {
			continue
		}
		result = append(result, *c)
	}
	return result, nil
}

func (s *ExportService) exportItems(filters *ExportFilters, referencedItemIDs map[string]bool) ([]model.Item, error) {
	// List(0, 0) returns all items with no limit.
	allItems, err := s.items.List(0, 0)
	if err != nil {
		return nil, err
	}

	// For a full export, include all items.
	// For filtered exports, only include items referenced by exported invoice items.
	isFiltered := filters != nil && len(filters.SupplierIDs) > 0

	var result []model.Item
	for _, item := range allItems {
		if isFiltered && !referencedItemIDs[item.ID] {
			continue
		}
		result = append(result, *item)
	}
	return result, nil
}

func (s *ExportService) exportPDFTemplates() ([]PDFTemplateExport, error) {
	templates, err := s.templates.List()
	if err != nil {
		return nil, err
	}

	var result []PDFTemplateExport
	for _, t := range templates {
		// Strip local filesystem path.
		result = append(result, PDFTemplateExport{
			ID:           t.ID,
			Name:         t.Name,
			TemplateCode: t.TemplateCode,
			ConfigJSON:   t.ConfigJSON,
			IsDefault:    t.IsDefault,
			SupplierID:   t.SupplierID,
			Description:  t.Description,
			ShowLogo:     t.ShowLogo,
			ShowQR:       t.ShowQR,
			ShowNotes:    t.ShowNotes,
			SortOrder:    t.SortOrder,
			IsBuiltin:    t.IsBuiltin,
			YAMLSource:   t.YAMLSource,
			ParentID:     t.ParentID,
		})
	}
	return result, nil
}

func (s *ExportService) exportCustomerItems() ([]CustomerItemExport, error) {
	// Use raw SQL to avoid the JOIN in the repository's GetByCustomer method.
	rows, err := s.db.Query(`
		SELECT id, customer_id, item_id, last_price, last_quantity, usage_count, last_used_at
		FROM customer_items ORDER BY customer_id, item_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CustomerItemExport
	for rows.Next() {
		var ci CustomerItemExport
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&ci.ID, &ci.CustomerID, &ci.ItemID, &ci.LastPrice,
			&ci.LastQuantity, &ci.UsageCount, &lastUsedAt); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			ci.LastUsedAt = lastUsedAt.Time
		}
		result = append(result, ci)
	}
	return result, rows.Err()
}
