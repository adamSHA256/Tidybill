package backup

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/adamSHA256/tidybill/internal/model"
)

// ImportService reads an ExportFile and imports data into the local database.
type ImportService struct {
	db *sql.DB
}

// NewImportService creates a new ImportService backed by the given database.
func NewImportService(db *sql.DB) *ImportService {
	return &ImportService{db: db}
}

// Import reads a .tidybill file and imports data according to the given options.
// If the file is encrypted, opts.Passphrase must be set. If the file is encrypted
// and no passphrase is provided, an error is returned.
func (s *ImportService) Import(reader io.Reader, opts ImportOptions) (*ImportReport, error) {
	// Read all data so we can check for encryption.
	rawData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading backup file: %w", err)
	}

	// Check if the file is encrypted and decrypt if needed.
	jsonData := rawData
	if IsEncrypted(rawData) {
		if opts.Passphrase == "" {
			return nil, errors.New("file is encrypted, passphrase required")
		}
		decrypted, err := DecryptExport(rawData, opts.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		jsonData = decrypted
	}

	// 1. Decode JSON
	var file ExportFile
	if err := json.NewDecoder(bytes.NewReader(jsonData)).Decode(&file); err != nil {
		return nil, fmt.Errorf("invalid backup file: %w", err)
	}

	// 2. Validate format version
	if file.Metadata.FormatVersion > FormatVersion {
		return nil, fmt.Errorf("backup file format version %d is newer than supported version %d", file.Metadata.FormatVersion, FormatVersion)
	}

	report := &ImportReport{
		Mode:      opts.Mode,
		StartedAt: time.Now(),
		Details:   make(map[string]TableSummary),
	}

	if opts.InvoiceNumberConflict == "" {
		opts.InvoiceNumberConflict = "skip"
	}

	// 3. Execute based on mode
	switch opts.Mode {
	case ImportModePreview:
		report, err = s.preview(&file, report, opts)
	case ImportModeFullReplace:
		report, err = s.fullReplace(&file, report, opts)
	case ImportModeSmartMerge:
		report, err = s.smartMerge(&file, report, opts)
	case ImportModeForce:
		report, err = s.forceImport(&file, report, opts)
	default:
		return nil, fmt.Errorf("unknown import mode: %s", opts.Mode)
	}

	if err != nil {
		return nil, err
	}

	report.FinishedAt = time.Now()
	report.Summary = summarize(report.Details, len(report.Conflicts), len(report.Warnings))
	return report, nil
}

// summarize aggregates per-table counts into the top-level summary.
func summarize(details map[string]TableSummary, conflicts, warnings int) ImportSummary {
	var s ImportSummary
	for _, t := range details {
		s.ToInsert += t.Insert
		s.ToUpdate += t.Update
		s.ToSkip += t.Skip
	}
	s.Conflicts = conflicts
	s.Warnings = warnings
	return s
}

// ---------------------------------------------------------------------------
// Full Replace
// ---------------------------------------------------------------------------

func (s *ImportService) fullReplace(file *ExportFile, report *ImportReport, opts ImportOptions) (*ImportReport, error) {
	err := withTx(s.db, func(tx *sql.Tx) error {
		// Delete ALL data in reverse FK order
		deleteTables := []string{
			"invoice_items", "invoices", "customer_items",
			"items", "pdf_templates", "smtp_configs",
			"bank_accounts", "customers", "suppliers",
			"settings", "vat_rates",
		}
		for _, table := range deleteTables {
			if _, err := tx.Exec("DELETE FROM " + table); err != nil {
				return fmt.Errorf("delete %s: %w", table, err)
			}
		}

		// Insert ALL data in FK order. abortOnError=true so any failure
		// rolls back the transaction instead of silently losing data.
		type tableInsert struct {
			table string
			fn    func() (TableSummary, error)
		}
		inserts := []tableInsert{
			{"vat_rates", func() (TableSummary, error) { return insertVatRates(tx, file.VatRates, true) }},
			{"settings", func() (TableSummary, error) { return insertSettings(tx, file.Settings, true) }},
			{"suppliers", func() (TableSummary, error) { return insertSuppliers(tx, file.Suppliers, true) }},
			{"bank_accounts", func() (TableSummary, error) { return insertBankAccounts(tx, file.BankAccounts, true) }},
			{"customers", func() (TableSummary, error) { return insertCustomers(tx, file.Customers, true) }},
			{"pdf_templates", func() (TableSummary, error) { return insertPDFTemplates(tx, file.PDFTemplates, true) }},
			{"items", func() (TableSummary, error) { return insertItems(tx, file.Items, true) }},
			{"smtp_configs", func() (TableSummary, error) { return insertSmtpConfigs(tx, file.SmtpConfigs, true) }},
			{"customer_items", func() (TableSummary, error) { return insertCustomerItems(tx, file.CustomerItems, true) }},
			{"invoices", func() (TableSummary, error) { return insertInvoices(tx, file.Invoices, true) }},
			{"invoice_items", func() (TableSummary, error) { return insertInvoiceItems(tx, file.InvoiceItems, true) }},
		}
		for _, ti := range inserts {
			ts, insertErr := ti.fn()
			report.Details[ti.table] = ts
			if insertErr != nil {
				return fmt.Errorf("insert %s: %w", ti.table, insertErr)
			}
		}

		// Resolve is_default flags
		if err := resolveDefaults(tx); err != nil {
			return fmt.Errorf("resolve defaults: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("full replace: %w", err)
	}

	// WAL checkpoint after commit
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return report, nil
}

// ---------------------------------------------------------------------------
// Smart Merge
// ---------------------------------------------------------------------------

func (s *ImportService) smartMerge(file *ExportFile, report *ImportReport, opts ImportOptions) (*ImportReport, error) {
	err := withTx(s.db, func(tx *sql.Tx) error {
		report.Details["vat_rates"] = mergeVatRates(tx, file.VatRates)
		report.Details["settings"] = mergeSettings(tx, file.Settings)
		report.Details["suppliers"] = mergeSuppliers(tx, file.Suppliers)
		report.Details["bank_accounts"] = mergeBankAccounts(tx, file.BankAccounts)
		report.Details["customers"] = mergeCustomers(tx, file.Customers)
		report.Details["pdf_templates"] = mergePDFTemplates(tx, file.PDFTemplates)
		report.Details["items"] = mergeItems(tx, file.Items)
		report.Details["smtp_configs"] = mergeSmtpConfigs(tx, file.SmtpConfigs)
		report.Details["customer_items"] = mergeCustomerItems(tx, file.CustomerItems)

		// Build invoice_items lookup by invoice_id for re-inserting items on invoice update.
		invoiceItemsByInvoice := make(map[string][]model.InvoiceItem)
		for _, ii := range file.InvoiceItems {
			invoiceItemsByInvoice[ii.InvoiceID] = append(invoiceItemsByInvoice[ii.InvoiceID], ii)
		}

		invSummary, conflicts, warnings := mergeInvoices(tx, file.Invoices, opts.InvoiceNumberConflict, invoiceItemsByInvoice)
		report.Details["invoices"] = invSummary
		report.Conflicts = append(report.Conflicts, conflicts...)
		report.Warnings = append(report.Warnings, warnings...)

		iiSummary, iiWarnings := mergeInvoiceItems(tx, file.InvoiceItems)
		report.Details["invoice_items"] = iiSummary
		report.Warnings = append(report.Warnings, iiWarnings...)

		if err := resolveDefaults(tx); err != nil {
			return fmt.Errorf("resolve defaults: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("smart merge: %w", err)
	}

	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return report, nil
}

// ---------------------------------------------------------------------------
// Force Import
// ---------------------------------------------------------------------------

func (s *ImportService) forceImport(file *ExportFile, report *ImportReport, opts ImportOptions) (*ImportReport, error) {
	err := withTx(s.db, func(tx *sql.Tx) error {
		report.Details["vat_rates"] = forceVatRates(tx, file.VatRates)
		report.Details["settings"] = mergeSettings(tx, file.Settings) // settings are always key-merge
		report.Details["suppliers"] = forceSuppliers(tx, file.Suppliers)
		report.Details["bank_accounts"] = forceBankAccounts(tx, file.BankAccounts)
		report.Details["customers"] = forceCustomers(tx, file.Customers)
		report.Details["pdf_templates"] = forcePDFTemplates(tx, file.PDFTemplates)
		report.Details["items"] = forceItems(tx, file.Items)
		report.Details["smtp_configs"] = forceSmtpConfigs(tx, file.SmtpConfigs)
		report.Details["customer_items"] = forceCustomerItems(tx, file.CustomerItems)

		invSummary, conflicts, warnings := forceInvoices(tx, file.Invoices, opts.InvoiceNumberConflict)
		report.Details["invoices"] = invSummary
		report.Conflicts = append(report.Conflicts, conflicts...)
		report.Warnings = append(report.Warnings, warnings...)

		report.Details["invoice_items"] = forceInvoiceItemsBatch(tx, file.InvoiceItems)

		if err := resolveDefaults(tx); err != nil {
			return fmt.Errorf("resolve defaults: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("force import: %w", err)
	}

	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return report, nil
}

// ---------------------------------------------------------------------------
// Preview (dry run)
// ---------------------------------------------------------------------------

func (s *ImportService) preview(file *ExportFile, report *ImportReport, opts ImportOptions) (*ImportReport, error) {
	// Use a read-only analysis: begin a transaction, do the counting, then rollback.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("preview: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // always rollback for preview

	simMode := opts.PreviewMode
	if simMode == "" || simMode == "merge" {
		simMode = ImportModeSmartMerge
	} else if simMode == "replace" {
		simMode = ImportModeFullReplace
	}
	// "force" stays as-is

	switch simMode {
	case ImportModeForce:
		// Force: all existing records will be updated, new ones inserted
		report.Details["vat_rates"] = previewForceByID(tx, "vat_rates", idsFromVatRates(file.VatRates))
		report.Details["settings"] = previewSettings(tx, file.Settings) // settings always key-merge
		report.Details["suppliers"] = previewForceByID(tx, "suppliers", idsFromSuppliers(file.Suppliers))
		report.Details["bank_accounts"] = previewForceByID(tx, "bank_accounts", idsFromBankAccounts(file.BankAccounts))
		report.Details["customers"] = previewForceByID(tx, "customers", idsFromCustomers(file.Customers))
		report.Details["pdf_templates"] = previewForceByID(tx, "pdf_templates", idsFromPDFTemplates(file.PDFTemplates))
		report.Details["items"] = previewForceByID(tx, "items", idsFromItems(file.Items))
		report.Details["smtp_configs"] = previewForceByID(tx, "smtp_configs", idsFromSmtpConfigs(file.SmtpConfigs))
		report.Details["customer_items"] = previewForceByID(tx, "customer_items", idsFromCustomerItems(file.CustomerItems))

		invSummary, conflicts := previewForceInvoices(tx, file.Invoices)
		report.Details["invoices"] = invSummary
		report.Conflicts = append(report.Conflicts, conflicts...)

		report.Details["invoice_items"] = previewForceByID(tx, "invoice_items", idsFromInvoiceItems(file.InvoiceItems))

	case ImportModeFullReplace:
		// Full replace: everything will be inserted (since all local data is deleted first)
		report.Details["vat_rates"] = TableSummary{Insert: len(file.VatRates)}
		report.Details["settings"] = TableSummary{Insert: len(file.Settings)}
		report.Details["suppliers"] = TableSummary{Insert: len(file.Suppliers)}
		report.Details["bank_accounts"] = TableSummary{Insert: len(file.BankAccounts)}
		report.Details["customers"] = TableSummary{Insert: len(file.Customers)}
		report.Details["pdf_templates"] = TableSummary{Insert: len(file.PDFTemplates)}
		report.Details["items"] = TableSummary{Insert: len(file.Items)}
		report.Details["smtp_configs"] = TableSummary{Insert: len(file.SmtpConfigs)}
		report.Details["customer_items"] = TableSummary{Insert: len(file.CustomerItems)}
		report.Details["invoices"] = TableSummary{Insert: len(file.Invoices)}
		report.Details["invoice_items"] = TableSummary{Insert: len(file.InvoiceItems)}

	default:
		// Smart merge (default preview behavior)
		report.Details["vat_rates"] = previewByID(tx, "vat_rates", idsFromVatRates(file.VatRates))
		report.Details["settings"] = previewSettings(tx, file.Settings)
		report.Details["suppliers"] = previewByIDWithTimestamp(tx, "suppliers", idsAndTimestamps(file.Suppliers))
		report.Details["bank_accounts"] = previewByID(tx, "bank_accounts", idsFromBankAccounts(file.BankAccounts))
		report.Details["customers"] = previewByIDWithTimestamp(tx, "customers", idsAndTimestampsCustomers(file.Customers))
		report.Details["pdf_templates"] = previewByID(tx, "pdf_templates", idsFromPDFTemplates(file.PDFTemplates))
		report.Details["items"] = previewByIDWithTimestamp(tx, "items", idsAndTimestampsItems(file.Items))
		report.Details["smtp_configs"] = previewByIDWithTimestamp(tx, "smtp_configs", idsAndTimestampsSmtp(file.SmtpConfigs))
		report.Details["customer_items"] = previewByID(tx, "customer_items", idsFromCustomerItems(file.CustomerItems))

		invSummary, conflicts := previewInvoices(tx, file.Invoices)
		report.Details["invoices"] = invSummary
		report.Conflicts = append(report.Conflicts, conflicts...)

		report.Details["invoice_items"] = previewByID(tx, "invoice_items", idsFromInvoiceItems(file.InvoiceItems))
	}

	report.FinishedAt = time.Now()
	report.Summary = summarize(report.Details, len(report.Conflicts), len(report.Warnings))
	return report, nil
}

// ---------------------------------------------------------------------------
// Transaction helper (mirrors repository.WithTx)
// ---------------------------------------------------------------------------

func withTx(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Full-replace INSERT helpers
// ---------------------------------------------------------------------------

func insertVatRates(tx *sql.Tx, rates []VatRateExport, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, r := range rates {
		_, err := tx.Exec(`INSERT INTO vat_rates (id, rate, name, is_default, country) VALUES (?, ?, ?, ?, ?)`,
			r.ID, r.Rate, r.Name, r.IsDefault, r.Country)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("vat_rate %s: %w", r.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertSettings(tx *sql.Tx, settings []SettingEntry, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, s := range settings {
		_, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, s.Key, s.Value)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("setting %s: %w", s.Key, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertSuppliers(tx *sql.Tx, suppliers []model.Supplier, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, s := range suppliers {
		_, err := tx.Exec(`INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic, ic_dph,
			phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language,
			created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.ICDPH,
			s.Phone, s.Email, s.Website, "", boolToInt(s.IsVATPayer), boolToInt(s.IsDefault),
			s.InvoicePrefix, s.Notes, s.Language, s.CreatedAt, s.UpdatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("supplier %s: %w", s.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertBankAccounts(tx *sql.Tx, accounts []model.BankAccount, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, ba := range accounts {
		_, err := tx.Exec(`INSERT INTO bank_accounts (id, supplier_id, name, account_number, iban, swift,
			currency, is_default, qr_type, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ba.ID, ba.SupplierID, ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT,
			ba.Currency, boolToInt(ba.IsDefault), ba.QRType, ba.CreatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("bank_account %s: %w", ba.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertCustomers(tx *sql.Tx, customers []model.Customer, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, c := range customers {
		_, err := tx.Exec(`INSERT INTO customers (id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes,
			email_custom_template, email_subject_template, email_body_template, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
			c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes,
			boolToInt(c.EmailCustomTemplate), c.EmailSubjectTemplate, c.EmailBodyTemplate,
			c.CreatedAt, c.UpdatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("customer %s: %w", c.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertPDFTemplates(tx *sql.Tx, templates []PDFTemplateExport, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, t := range templates {
		_, err := tx.Exec(`INSERT INTO pdf_templates (id, name, template_code, config_json, is_default,
			supplier_id, description, show_logo, show_qr, show_notes, sort_order, is_builtin, yaml_source, parent_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.Name, t.TemplateCode, t.ConfigJSON, boolToInt(t.IsDefault),
			nullString(t.SupplierID), t.Description, boolToInt(t.ShowLogo), boolToInt(t.ShowQR),
			boolToInt(t.ShowNotes), t.SortOrder, boolToInt(t.IsBuiltin), t.YAMLSource, t.ParentID)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("pdf_template %s: %w", t.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertItems(tx *sql.Tx, items []model.Item, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, item := range items {
		_, err := tx.Exec(`INSERT INTO items (id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
			item.Category, item.LastUsedPrice, nullString(item.LastCustomerID),
			item.UsageCount, item.CreatedAt, item.UpdatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("item %s: %w", item.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertSmtpConfigs(tx *sql.Tx, configs []SmtpConfigExport, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, c := range configs {
		// Never import passwords; password_encrypted is always empty string on import.
		_, err := tx.Exec(`INSERT INTO smtp_configs (id, supplier_id, host, port, username,
			password_encrypted, from_name, from_email, use_starttls, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?)`,
			c.ID, c.SupplierID, c.Host, c.Port, c.Username,
			c.FromName, c.FromEmail, boolToInt(c.UseStartTLS), boolToInt(c.Enabled),
			c.CreatedAt, c.UpdatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("smtp_config %s: %w", c.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertCustomerItems(tx *sql.Tx, items []CustomerItemExport, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, ci := range items {
		_, err := tx.Exec(`INSERT INTO customer_items (id, customer_id, item_id, last_price, last_quantity,
			usage_count, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ci.ID, ci.CustomerID, ci.ItemID, ci.LastPrice, ci.LastQuantity,
			ci.UsageCount, ci.LastUsedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("customer_item %s: %w", ci.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertInvoices(tx *sql.Tx, invoices []InvoiceExport, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, inv := range invoices {
		_, err := tx.Exec(`INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id,
			status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
			currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
			email_sent_at, language, pdf_path, template_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)`,
			inv.ID, inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID,
			inv.Status, inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate,
			inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
			inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
			inv.EmailSentAt, inv.Language, inv.TemplateID, inv.CreatedAt, inv.UpdatedAt)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("invoice %s: %w", inv.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

func insertInvoiceItems(tx *sql.Tx, items []model.InvoiceItem, abortOnError bool) (TableSummary, error) {
	var ts TableSummary
	for _, item := range items {
		_, err := tx.Exec(`INSERT INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.InvoiceID, nullString(item.ItemID), item.Description, item.Quantity,
			item.Unit, item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount,
			item.Total, item.Position)
		if err == nil {
			ts.Insert++
		} else if abortOnError {
			return ts, fmt.Errorf("invoice_item %s: %w", item.ID, err)
		} else {
			ts.Skip++
		}
	}
	return ts, nil
}

// ---------------------------------------------------------------------------
// Smart Merge helpers
// ---------------------------------------------------------------------------

// mergeVatRates: insert if ID missing, skip if exists (no updated_at).
func mergeVatRates(tx *sql.Tx, rates []VatRateExport) TableSummary {
	var ts TableSummary
	for _, r := range rates {
		if existsByID(tx, "vat_rates", r.ID) {
			ts.Skip++
			continue
		}
		_, err := tx.Exec(`INSERT INTO vat_rates (id, rate, name, is_default, country) VALUES (?, ?, ?, ?, ?)`,
			r.ID, r.Rate, r.Name, r.IsDefault, r.Country)
		if err == nil {
			ts.Insert++
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeSettings: key-value merge. Each key from import overwrites local. Local-only keys are kept.
func mergeSettings(tx *sql.Tx, settings []SettingEntry) TableSummary {
	var ts TableSummary
	for _, s := range settings {
		var existing string
		err := tx.QueryRow("SELECT key FROM settings WHERE key = ?", s.Key).Scan(&existing)
		if err == sql.ErrNoRows {
			_, err = tx.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", s.Key, s.Value)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		} else if err == nil {
			_, err = tx.Exec("UPDATE settings SET value = ? WHERE key = ?", s.Value, s.Key)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeSuppliers: compare updated_at, keep newer.
func mergeSuppliers(tx *sql.Tx, suppliers []model.Supplier) TableSummary {
	var ts TableSummary
	for _, s := range suppliers {
		localUpdatedAt, exists := getUpdatedAt(tx, "suppliers", s.ID)
		if !exists {
			_, err := tx.Exec(`INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic, ic_dph,
				phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language,
				created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				s.ID, s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.ICDPH,
				s.Phone, s.Email, s.Website, "", boolToInt(s.IsVATPayer), boolToInt(s.IsDefault),
				s.InvoicePrefix, s.Notes, s.Language, s.CreatedAt, s.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		} else if s.UpdatedAt.After(localUpdatedAt) {
			_, err := tx.Exec(`UPDATE suppliers SET name=?, street=?, city=?, zip=?, country=?, ico=?, dic=?, ic_dph=?,
				phone=?, email=?, website=?, is_vat_payer=?, is_default=?, invoice_prefix=?, notes=?, language=?,
				updated_at=? WHERE id=?`,
				s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.ICDPH,
				s.Phone, s.Email, s.Website, boolToInt(s.IsVATPayer), boolToInt(s.IsDefault),
				s.InvoicePrefix, s.Notes, s.Language, s.UpdatedAt, s.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeBankAccounts: no updated_at, so skip if exists locally.
func mergeBankAccounts(tx *sql.Tx, accounts []model.BankAccount) TableSummary {
	var ts TableSummary
	for _, ba := range accounts {
		if existsByID(tx, "bank_accounts", ba.ID) {
			ts.Skip++
			continue
		}
		_, err := tx.Exec(`INSERT INTO bank_accounts (id, supplier_id, name, account_number, iban, swift,
			currency, is_default, qr_type, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ba.ID, ba.SupplierID, ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT,
			ba.Currency, boolToInt(ba.IsDefault), ba.QRType, ba.CreatedAt)
		if err == nil {
			ts.Insert++
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeCustomers: compare updated_at, keep newer.
func mergeCustomers(tx *sql.Tx, customers []model.Customer) TableSummary {
	var ts TableSummary
	for _, c := range customers {
		localUpdatedAt, exists := getUpdatedAt(tx, "customers", c.ID)
		if !exists {
			_, err := tx.Exec(`INSERT INTO customers (id, name, street, city, zip, region, country, ico, dic, ic_dph,
				email, phone, default_vat_rate, default_due_days, notes,
				email_custom_template, email_subject_template, email_body_template, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				c.ID, c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
				c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes,
				boolToInt(c.EmailCustomTemplate), c.EmailSubjectTemplate, c.EmailBodyTemplate,
				c.CreatedAt, c.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		} else if c.UpdatedAt.After(localUpdatedAt) {
			_, err := tx.Exec(`UPDATE customers SET name=?, street=?, city=?, zip=?, region=?, country=?, ico=?, dic=?, ic_dph=?,
				email=?, phone=?, default_vat_rate=?, default_due_days=?, notes=?,
				email_custom_template=?, email_subject_template=?, email_body_template=?, updated_at=?
				WHERE id=?`,
				c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
				c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes,
				boolToInt(c.EmailCustomTemplate), c.EmailSubjectTemplate, c.EmailBodyTemplate,
				c.UpdatedAt, c.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergePDFTemplates: no updated_at, skip if exists.
func mergePDFTemplates(tx *sql.Tx, templates []PDFTemplateExport) TableSummary {
	var ts TableSummary
	for _, t := range templates {
		if existsByID(tx, "pdf_templates", t.ID) {
			ts.Skip++
			continue
		}
		_, err := tx.Exec(`INSERT INTO pdf_templates (id, name, template_code, config_json, is_default,
			supplier_id, description, show_logo, show_qr, show_notes, sort_order, is_builtin, yaml_source, parent_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.Name, t.TemplateCode, t.ConfigJSON, boolToInt(t.IsDefault),
			nullString(t.SupplierID), t.Description, boolToInt(t.ShowLogo), boolToInt(t.ShowQR),
			boolToInt(t.ShowNotes), t.SortOrder, boolToInt(t.IsBuiltin), t.YAMLSource, t.ParentID)
		if err == nil {
			ts.Insert++
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeItems: compare updated_at, keep newer.
func mergeItems(tx *sql.Tx, items []model.Item) TableSummary {
	var ts TableSummary
	for _, item := range items {
		localUpdatedAt, exists := getUpdatedAt(tx, "items", item.ID)
		if !exists {
			_, err := tx.Exec(`INSERT INTO items (id, description, default_price, default_unit, default_vat_rate,
				category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				item.ID, item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
				item.Category, item.LastUsedPrice, nullString(item.LastCustomerID),
				item.UsageCount, item.CreatedAt, item.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		} else if item.UpdatedAt.After(localUpdatedAt) {
			_, err := tx.Exec(`UPDATE items SET description=?, default_price=?, default_unit=?, default_vat_rate=?,
				category=?, last_used_price=?, last_customer_id=?, usage_count=?, updated_at=?
				WHERE id=?`,
				item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
				item.Category, item.LastUsedPrice, nullString(item.LastCustomerID),
				item.UsageCount, item.UpdatedAt, item.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeSmtpConfigs: compare updated_at, keep newer. Never import passwords.
func mergeSmtpConfigs(tx *sql.Tx, configs []SmtpConfigExport) TableSummary {
	var ts TableSummary
	for _, c := range configs {
		localUpdatedAt, exists := getUpdatedAt(tx, "smtp_configs", c.ID)
		if !exists {
			_, err := tx.Exec(`INSERT INTO smtp_configs (id, supplier_id, host, port, username,
				password_encrypted, from_name, from_email, use_starttls, enabled, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?)`,
				c.ID, c.SupplierID, c.Host, c.Port, c.Username,
				c.FromName, c.FromEmail, boolToInt(c.UseStartTLS), boolToInt(c.Enabled),
				c.CreatedAt, c.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		} else if c.UpdatedAt.After(localUpdatedAt) {
			// Update everything except password_encrypted
			_, err := tx.Exec(`UPDATE smtp_configs SET host=?, port=?, username=?,
				from_name=?, from_email=?, use_starttls=?, enabled=?, updated_at=?
				WHERE id=?`,
				c.Host, c.Port, c.Username,
				c.FromName, c.FromEmail, boolToInt(c.UseStartTLS), boolToInt(c.Enabled),
				c.UpdatedAt, c.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeCustomerItems: no updated_at, skip if exists.
func mergeCustomerItems(tx *sql.Tx, items []CustomerItemExport) TableSummary {
	var ts TableSummary
	for _, ci := range items {
		if existsByID(tx, "customer_items", ci.ID) {
			ts.Skip++
			continue
		}
		_, err := tx.Exec(`INSERT INTO customer_items (id, customer_id, item_id, last_price, last_quantity,
			usage_count, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ci.ID, ci.CustomerID, ci.ItemID, ci.LastPrice, ci.LastQuantity,
			ci.UsageCount, ci.LastUsedAt)
		if err == nil {
			ts.Insert++
		} else {
			ts.Skip++
		}
	}
	return ts
}

// mergeInvoices: compare updated_at, keep newer. Handle invoice_number collisions.
// When an invoice is updated, its invoice_items are also replaced with the remote versions.
func mergeInvoices(tx *sql.Tx, invoices []InvoiceExport, conflictMode string, invoiceItemsByInvoice map[string][]model.InvoiceItem) (TableSummary, []ImportConflict, []ImportWarning) {
	var ts TableSummary
	var conflicts []ImportConflict
	var warnings []ImportWarning

	for _, inv := range invoices {
		localUpdatedAt, exists := getUpdatedAt(tx, "invoices", inv.ID)
		if exists {
			// UUID exists locally, compare timestamps
			if inv.UpdatedAt.After(localUpdatedAt) {
				_, err := tx.Exec(`UPDATE invoices SET invoice_number=?, supplier_id=?, customer_id=?, bank_account_id=?, status=?,
					issue_date=?, due_date=?, paid_date=?, taxable_date=?, payment_method=?, variable_symbol=?,
					currency=?, exchange_rate=?, subtotal=?, vat_total=?, total=?, notes=?, internal_notes=?,
					email_sent_at=?, language=?, template_id=?, updated_at=?
					WHERE id=?`,
					inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID, inv.Status,
					inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate,
					inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
					inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
					inv.EmailSentAt, inv.Language, inv.TemplateID, inv.UpdatedAt, inv.ID)
				if err == nil {
					ts.Update++
					// Re-insert invoice_items for the updated invoice to ensure consistency.
					if items, ok := invoiceItemsByInvoice[inv.ID]; ok {
						tx.Exec("DELETE FROM invoice_items WHERE invoice_id = ?", inv.ID) //nolint:errcheck
						for _, item := range items {
							tx.Exec(`INSERT OR REPLACE INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
								unit_price, vat_rate, subtotal, vat_amount, total, position)
								VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
								item.ID, item.InvoiceID, nullString(item.ItemID), item.Description, item.Quantity,
								item.Unit, item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount,
								item.Total, item.Position) //nolint:errcheck
						}
					}
				} else {
					ts.Skip++
				}
			} else {
				ts.Skip++
			}
		} else {
			// UUID doesn't exist; check for invoice_number collision
			collision := checkInvoiceNumberCollision(tx, inv.ID, inv.SupplierID, inv.InvoiceNumber)
			if collision {
				conflicts = append(conflicts, ImportConflict{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "invoice_number_collision",
					Description: fmt.Sprintf("Invoice number %s already exists for supplier %s (different invoice ID)", inv.InvoiceNumber, inv.SupplierID),
					Resolution:  conflictMode,
				})
				if conflictMode == "auto_suffix" {
					inv.InvoiceNumber = inv.InvoiceNumber + "-imp"
					inv.VariableSymbol = inv.VariableSymbol + "9"
				} else {
					ts.Skip++
					continue
				}
			}

			_, err := tx.Exec(`INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id,
				status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
				currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
				email_sent_at, language, pdf_path, template_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)`,
				inv.ID, inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID,
				inv.Status, inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate,
				inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
				inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
				inv.EmailSentAt, inv.Language, inv.TemplateID, inv.CreatedAt, inv.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				warnings = append(warnings, ImportWarning{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "insert_failed",
					Description: fmt.Sprintf("Failed to insert invoice %s: %v", inv.InvoiceNumber, err),
					Resolution:  "skipped",
				})
				ts.Skip++
			}
		}
	}
	return ts, conflicts, warnings
}

// mergeInvoiceItems: no updated_at, skip if exists. Validate invoice FK.
func mergeInvoiceItems(tx *sql.Tx, items []model.InvoiceItem) (TableSummary, []ImportWarning) {
	var ts TableSummary
	var warnings []ImportWarning
	for _, item := range items {
		if existsByID(tx, "invoice_items", item.ID) {
			ts.Skip++
			continue
		}
		// Validate FK: invoice must exist
		if !existsByID(tx, "invoices", item.InvoiceID) {
			warnings = append(warnings, ImportWarning{
				Table:       "invoice_items",
				ID:          item.ID,
				Type:        "missing_fk",
				Description: fmt.Sprintf("invoice_id %s not found locally or in import", item.InvoiceID),
				Resolution:  "skipped",
			})
			ts.Skip++
			continue
		}
		_, err := tx.Exec(`INSERT INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.InvoiceID, nullString(item.ItemID), item.Description, item.Quantity,
			item.Unit, item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount,
			item.Total, item.Position)
		if err == nil {
			ts.Insert++
		} else {
			ts.Skip++
		}
	}
	return ts, warnings
}

// ---------------------------------------------------------------------------
// Force Import helpers (INSERT OR REPLACE, no timestamp check)
// ---------------------------------------------------------------------------

func forceVatRates(tx *sql.Tx, rates []VatRateExport) TableSummary {
	var ts TableSummary
	for _, r := range rates {
		existed := existsByID(tx, "vat_rates", r.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO vat_rates (id, rate, name, is_default, country) VALUES (?, ?, ?, ?, ?)`,
			r.ID, r.Rate, r.Name, r.IsDefault, r.Country)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceSuppliers(tx *sql.Tx, suppliers []model.Supplier) TableSummary {
	var ts TableSummary
	for _, s := range suppliers {
		existed := existsByID(tx, "suppliers", s.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO suppliers (id, name, street, city, zip, country, ico, dic, ic_dph,
			phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language,
			created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.ICDPH,
			s.Phone, s.Email, s.Website, "", boolToInt(s.IsVATPayer), boolToInt(s.IsDefault),
			s.InvoicePrefix, s.Notes, s.Language, s.CreatedAt, s.UpdatedAt)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceBankAccounts(tx *sql.Tx, accounts []model.BankAccount) TableSummary {
	var ts TableSummary
	for _, ba := range accounts {
		existed := existsByID(tx, "bank_accounts", ba.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO bank_accounts (id, supplier_id, name, account_number, iban, swift,
			currency, is_default, qr_type, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ba.ID, ba.SupplierID, ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT,
			ba.Currency, boolToInt(ba.IsDefault), ba.QRType, ba.CreatedAt)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceCustomers(tx *sql.Tx, customers []model.Customer) TableSummary {
	var ts TableSummary
	for _, c := range customers {
		existed := existsByID(tx, "customers", c.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO customers (id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes,
			email_custom_template, email_subject_template, email_body_template, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
			c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes,
			boolToInt(c.EmailCustomTemplate), c.EmailSubjectTemplate, c.EmailBodyTemplate,
			c.CreatedAt, c.UpdatedAt)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forcePDFTemplates(tx *sql.Tx, templates []PDFTemplateExport) TableSummary {
	var ts TableSummary
	for _, t := range templates {
		existed := existsByID(tx, "pdf_templates", t.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO pdf_templates (id, name, template_code, config_json, is_default,
			supplier_id, description, show_logo, show_qr, show_notes, sort_order, is_builtin, yaml_source, parent_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.Name, t.TemplateCode, t.ConfigJSON, boolToInt(t.IsDefault),
			nullString(t.SupplierID), t.Description, boolToInt(t.ShowLogo), boolToInt(t.ShowQR),
			boolToInt(t.ShowNotes), t.SortOrder, boolToInt(t.IsBuiltin), t.YAMLSource, t.ParentID)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceItems(tx *sql.Tx, items []model.Item) TableSummary {
	var ts TableSummary
	for _, item := range items {
		existed := existsByID(tx, "items", item.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO items (id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
			item.Category, item.LastUsedPrice, nullString(item.LastCustomerID),
			item.UsageCount, item.CreatedAt, item.UpdatedAt)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceSmtpConfigs(tx *sql.Tx, configs []SmtpConfigExport) TableSummary {
	var ts TableSummary
	for _, c := range configs {
		existed := existsByID(tx, "smtp_configs", c.ID)
		if existed {
			// Update everything except password_encrypted
			_, err := tx.Exec(`UPDATE smtp_configs SET supplier_id=?, host=?, port=?, username=?,
				from_name=?, from_email=?, use_starttls=?, enabled=?, updated_at=?
				WHERE id=?`,
				c.SupplierID, c.Host, c.Port, c.Username,
				c.FromName, c.FromEmail, boolToInt(c.UseStartTLS), boolToInt(c.Enabled),
				c.UpdatedAt, c.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			_, err := tx.Exec(`INSERT INTO smtp_configs (id, supplier_id, host, port, username,
				password_encrypted, from_name, from_email, use_starttls, enabled, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?)`,
				c.ID, c.SupplierID, c.Host, c.Port, c.Username,
				c.FromName, c.FromEmail, boolToInt(c.UseStartTLS), boolToInt(c.Enabled),
				c.CreatedAt, c.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				ts.Skip++
			}
		}
	}
	return ts
}

func forceCustomerItems(tx *sql.Tx, items []CustomerItemExport) TableSummary {
	var ts TableSummary
	for _, ci := range items {
		existed := existsByID(tx, "customer_items", ci.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO customer_items (id, customer_id, item_id, last_price, last_quantity,
			usage_count, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ci.ID, ci.CustomerID, ci.ItemID, ci.LastPrice, ci.LastQuantity,
			ci.UsageCount, ci.LastUsedAt)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

func forceInvoices(tx *sql.Tx, invoices []InvoiceExport, conflictMode string) (TableSummary, []ImportConflict, []ImportWarning) {
	var ts TableSummary
	var conflicts []ImportConflict
	var warnings []ImportWarning

	for _, inv := range invoices {
		existed := existsByID(tx, "invoices", inv.ID)
		if existed {
			_, err := tx.Exec(`UPDATE invoices SET invoice_number=?, supplier_id=?, customer_id=?, bank_account_id=?, status=?,
				issue_date=?, due_date=?, paid_date=?, taxable_date=?, payment_method=?, variable_symbol=?,
				currency=?, exchange_rate=?, subtotal=?, vat_total=?, total=?, notes=?, internal_notes=?,
				email_sent_at=?, language=?, template_id=?, updated_at=?
				WHERE id=?`,
				inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID, inv.Status,
				inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate,
				inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
				inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
				inv.EmailSentAt, inv.Language, inv.TemplateID, inv.UpdatedAt, inv.ID)
			if err == nil {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			// Check invoice_number collision
			collision := checkInvoiceNumberCollision(tx, inv.ID, inv.SupplierID, inv.InvoiceNumber)
			if collision {
				conflicts = append(conflicts, ImportConflict{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "invoice_number_collision",
					Description: fmt.Sprintf("Invoice number %s already exists for supplier %s (different invoice ID)", inv.InvoiceNumber, inv.SupplierID),
					Resolution:  conflictMode,
				})
				if conflictMode == "auto_suffix" {
					inv.InvoiceNumber = inv.InvoiceNumber + "-imp"
					inv.VariableSymbol = inv.VariableSymbol + "9"
				} else {
					ts.Skip++
					continue
				}
			}

			_, err := tx.Exec(`INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id,
				status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
				currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
				email_sent_at, language, pdf_path, template_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)`,
				inv.ID, inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID,
				inv.Status, inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate,
				inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
				inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
				inv.EmailSentAt, inv.Language, inv.TemplateID, inv.CreatedAt, inv.UpdatedAt)
			if err == nil {
				ts.Insert++
			} else {
				warnings = append(warnings, ImportWarning{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "insert_failed",
					Description: fmt.Sprintf("Failed to insert invoice %s: %v", inv.InvoiceNumber, err),
					Resolution:  "skipped",
				})
				ts.Skip++
			}
		}
	}
	return ts, conflicts, warnings
}

func forceInvoiceItemsBatch(tx *sql.Tx, items []model.InvoiceItem) TableSummary {
	var ts TableSummary
	for _, item := range items {
		existed := existsByID(tx, "invoice_items", item.ID)
		_, err := tx.Exec(`INSERT OR REPLACE INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.InvoiceID, nullString(item.ItemID), item.Description, item.Quantity,
			item.Unit, item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount,
			item.Total, item.Position)
		if err == nil {
			if existed {
				ts.Update++
			} else {
				ts.Insert++
			}
		} else {
			ts.Skip++
		}
	}
	return ts
}

// ---------------------------------------------------------------------------
// Preview helpers
// ---------------------------------------------------------------------------

type idTimestamp struct {
	id        string
	updatedAt time.Time
}

func previewByID(tx *sql.Tx, table string, ids []string) TableSummary {
	var ts TableSummary
	for _, id := range ids {
		if existsByID(tx, table, id) {
			ts.Skip++
		} else {
			ts.Insert++
		}
	}
	return ts
}

func previewByIDWithTimestamp(tx *sql.Tx, table string, entries []idTimestamp) TableSummary {
	var ts TableSummary
	for _, e := range entries {
		localUpdatedAt, exists := getUpdatedAt(tx, table, e.id)
		if !exists {
			ts.Insert++
		} else if e.updatedAt.After(localUpdatedAt) {
			ts.Update++
		} else {
			ts.Skip++
		}
	}
	return ts
}

func previewSettings(tx *sql.Tx, settings []SettingEntry) TableSummary {
	var ts TableSummary
	for _, s := range settings {
		var existing string
		err := tx.QueryRow("SELECT key FROM settings WHERE key = ?", s.Key).Scan(&existing)
		if err == sql.ErrNoRows {
			ts.Insert++
		} else {
			ts.Update++
		}
	}
	return ts
}

func previewInvoices(tx *sql.Tx, invoices []InvoiceExport) (TableSummary, []ImportConflict) {
	var ts TableSummary
	var conflicts []ImportConflict
	for _, inv := range invoices {
		localUpdatedAt, exists := getUpdatedAt(tx, "invoices", inv.ID)
		if exists {
			if inv.UpdatedAt.After(localUpdatedAt) {
				ts.Update++
			} else {
				ts.Skip++
			}
		} else {
			collision := checkInvoiceNumberCollision(tx, inv.ID, inv.SupplierID, inv.InvoiceNumber)
			if collision {
				conflicts = append(conflicts, ImportConflict{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "invoice_number_collision",
					Description: fmt.Sprintf("Invoice number %s already exists for supplier %s (different invoice ID)", inv.InvoiceNumber, inv.SupplierID),
					Resolution:  "skip",
				})
				ts.Skip++
			} else {
				ts.Insert++
			}
		}
	}
	return ts, conflicts
}

// previewForceByID simulates force import: existing records = update, new = insert.
func previewForceByID(tx *sql.Tx, table string, ids []string) TableSummary {
	var ts TableSummary
	for _, id := range ids {
		if existsByID(tx, table, id) {
			ts.Update++
		} else {
			ts.Insert++
		}
	}
	return ts
}

// previewForceInvoices simulates force import for invoices.
func previewForceInvoices(tx *sql.Tx, invoices []InvoiceExport) (TableSummary, []ImportConflict) {
	var ts TableSummary
	var conflicts []ImportConflict
	for _, inv := range invoices {
		if existsByID(tx, "invoices", inv.ID) {
			ts.Update++
		} else {
			collision := checkInvoiceNumberCollision(tx, inv.ID, inv.SupplierID, inv.InvoiceNumber)
			if collision {
				conflicts = append(conflicts, ImportConflict{
					Table:       "invoices",
					ID:          inv.ID,
					Type:        "invoice_number_collision",
					Description: fmt.Sprintf("Invoice number %s already exists for supplier %s (different invoice ID)", inv.InvoiceNumber, inv.SupplierID),
					Resolution:  "skip",
				})
				ts.Skip++
			} else {
				ts.Insert++
			}
		}
	}
	return ts, conflicts
}

// ID extraction helpers for preview
func idsFromVatRates(rates []VatRateExport) []string {
	ids := make([]string, len(rates))
	for i, r := range rates {
		ids[i] = r.ID
	}
	return ids
}

func idsFromSuppliers(suppliers []model.Supplier) []string {
	ids := make([]string, len(suppliers))
	for i, s := range suppliers {
		ids[i] = s.ID
	}
	return ids
}

func idsFromBankAccounts(accounts []model.BankAccount) []string {
	ids := make([]string, len(accounts))
	for i, a := range accounts {
		ids[i] = a.ID
	}
	return ids
}

func idsFromCustomers(customers []model.Customer) []string {
	ids := make([]string, len(customers))
	for i, c := range customers {
		ids[i] = c.ID
	}
	return ids
}

func idsFromPDFTemplates(templates []PDFTemplateExport) []string {
	ids := make([]string, len(templates))
	for i, t := range templates {
		ids[i] = t.ID
	}
	return ids
}

func idsFromCustomerItems(items []CustomerItemExport) []string {
	ids := make([]string, len(items))
	for i, ci := range items {
		ids[i] = ci.ID
	}
	return ids
}

func idsFromItems(items []model.Item) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

func idsFromSmtpConfigs(configs []SmtpConfigExport) []string {
	ids := make([]string, len(configs))
	for i, c := range configs {
		ids[i] = c.ID
	}
	return ids
}

func idsFromInvoiceItems(items []model.InvoiceItem) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

func idsAndTimestamps(suppliers []model.Supplier) []idTimestamp {
	entries := make([]idTimestamp, len(suppliers))
	for i, s := range suppliers {
		entries[i] = idTimestamp{id: s.ID, updatedAt: s.UpdatedAt}
	}
	return entries
}

func idsAndTimestampsCustomers(customers []model.Customer) []idTimestamp {
	entries := make([]idTimestamp, len(customers))
	for i, c := range customers {
		entries[i] = idTimestamp{id: c.ID, updatedAt: c.UpdatedAt}
	}
	return entries
}

func idsAndTimestampsItems(items []model.Item) []idTimestamp {
	entries := make([]idTimestamp, len(items))
	for i, item := range items {
		entries[i] = idTimestamp{id: item.ID, updatedAt: item.UpdatedAt}
	}
	return entries
}

func idsAndTimestampsSmtp(configs []SmtpConfigExport) []idTimestamp {
	entries := make([]idTimestamp, len(configs))
	for i, c := range configs {
		entries[i] = idTimestamp{id: c.ID, updatedAt: c.UpdatedAt}
	}
	return entries
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// existsByID checks whether a row with the given primary key exists in the table.
func existsByID(tx *sql.Tx, table, id string) bool {
	var count int
	//nolint:gosec // table name is always a compile-time constant from this package
	_ = tx.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&count)
	return count > 0
}

// getUpdatedAt returns the updated_at timestamp for a row, or false if not found.
func getUpdatedAt(tx *sql.Tx, table, id string) (time.Time, bool) {
	var t time.Time
	//nolint:gosec // table name is always a compile-time constant from this package
	err := tx.QueryRow("SELECT updated_at FROM "+table+" WHERE id = ?", id).Scan(&t)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// checkInvoiceNumberCollision returns true if the invoice_number already exists
// for the given supplier with a different invoice ID.
func checkInvoiceNumberCollision(tx *sql.Tx, invoiceID, supplierID, invoiceNumber string) bool {
	var count int
	_ = tx.QueryRow(
		`SELECT COUNT(*) FROM invoices WHERE supplier_id = ? AND invoice_number = ? AND id != ?`,
		supplierID, invoiceNumber, invoiceID,
	).Scan(&count)
	return count > 0
}

// resolveDefaults ensures exactly one default per scope after import.
func resolveDefaults(tx *sql.Tx) error {
	// 1. Suppliers: ensure exactly one is_default = true
	var defaultCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM suppliers WHERE is_default = 1").Scan(&defaultCount); err != nil {
		return err
	}
	if defaultCount == 0 {
		_, _ = tx.Exec("UPDATE suppliers SET is_default = 1 WHERE id = (SELECT id FROM suppliers ORDER BY created_at ASC LIMIT 1)")
	} else if defaultCount > 1 {
		var winnerID string
		if err := tx.QueryRow("SELECT id FROM suppliers WHERE is_default = 1 ORDER BY updated_at DESC LIMIT 1").Scan(&winnerID); err == nil {
			_, _ = tx.Exec("UPDATE suppliers SET is_default = 0")
			_, _ = tx.Exec("UPDATE suppliers SET is_default = 1 WHERE id = ?", winnerID)
		}
	}

	// 2. Bank accounts: ensure at most one is_default per supplier
	rows, err := tx.Query("SELECT supplier_id FROM bank_accounts GROUP BY supplier_id HAVING SUM(is_default) > 1")
	if err != nil {
		return err
	}
	var supplierIDs []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			return err
		}
		supplierIDs = append(supplierIDs, sid)
	}
	rows.Close()

	for _, sid := range supplierIDs {
		var winnerID string
		if err := tx.QueryRow("SELECT id FROM bank_accounts WHERE supplier_id = ? AND is_default = 1 ORDER BY created_at ASC LIMIT 1", sid).Scan(&winnerID); err == nil {
			_, _ = tx.Exec("UPDATE bank_accounts SET is_default = 0 WHERE supplier_id = ?", sid)
			_, _ = tx.Exec("UPDATE bank_accounts SET is_default = 1 WHERE id = ?", winnerID)
		}
	}

	// 3. PDF templates: ensure exactly one is_default = true
	if err := tx.QueryRow("SELECT COUNT(*) FROM pdf_templates WHERE is_default = 1").Scan(&defaultCount); err != nil {
		return err
	}
	if defaultCount == 0 {
		_, _ = tx.Exec("UPDATE pdf_templates SET is_default = 1 WHERE id = 'default'")
	} else if defaultCount > 1 {
		_, _ = tx.Exec("UPDATE pdf_templates SET is_default = 0")
		_, _ = tx.Exec("UPDATE pdf_templates SET is_default = 1 WHERE id = 'default'")
	}

	return nil
}

// boolToInt converts a Go bool to a SQLite integer (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullString returns nil for empty strings (for nullable FK columns).
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
