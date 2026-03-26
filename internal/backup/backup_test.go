package backup

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/model"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupTestDB creates an in-memory SQLite database with all migrations applied
// and returns the raw *sql.DB plus seed data IDs for use in tests.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Use a temp file-based DB since in-memory DBs have issues with multiple
	// connections seeing different databases, and single-connection mode
	// deadlocks on PRAGMA foreign_keys = OFF in multi-statement migrations.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	dsn := dbPath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// Run all migrations in order.
	runMigrations(t, db)

	return db
}

// findMigrationsDir locates the migrations directory by walking up from the
// current working directory. go test sets cwd to the package dir, but just
// in case, we also try known relative paths.
func findMigrationsDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "database", "migrations"),             // from internal/backup
		filepath.Join("internal", "database", "migrations"),      // from repo root
	}
	// Also try absolute path as fallback.
	absPath := filepath.Join(projectRoot(), "internal", "database", "migrations")
	candidates = append(candidates, absPath)

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	t.Fatalf("could not find migrations directory; tried: %v", candidates)
	return ""
}

// projectRoot returns the project root by looking for go.mod.
func projectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	migrationsDir := findMigrationsDir(t)

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir %s: %v", migrationsDir, err)
	}

	var migrations []fs.DirEntry
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			migrations = append(migrations, e)
		}
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name() < migrations[j].Name()
	})

	for _, m := range migrations {
		version := strings.TrimSuffix(m.Name(), ".sql")
		content, err := os.ReadFile(filepath.Join(migrationsDir, m.Name()))
		if err != nil {
			t.Fatalf("read migration %s: %v", m.Name(), err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("migration %s failed: %v", m.Name(), err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			t.Fatalf("record migration %s: %v", m.Name(), err)
		}
	}
}

type seedIDs struct {
	supplierID    string
	bankAccountID string
	customerID    string
	invoiceID     string
	invoiceItemID string
	itemID        string
	customerItemID string
	smtpConfigID  string
	templateID    string
}

// seedTestData inserts a minimal but complete set of data into the database
// and returns all generated IDs.
func seedTestData(t *testing.T, db *sql.DB) seedIDs {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	ids := seedIDs{
		supplierID:    "sup-test-001",
		bankAccountID: "ba-test-001",
		customerID:    "cust-test-001",
		invoiceID:     "inv-test-001",
		invoiceItemID: "ii-test-001",
		itemID:        "item-test-001",
		customerItemID: "ci-test-001",
		smtpConfigID:  "smtp-test-001",
		templateID:    "tpl-test-001",
	}

	// Supplier
	mustExec(t, db, `INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic, ic_dph,
		phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language,
		created_at, updated_at)
		VALUES (?, 'Test Supplier', 'Testovaci 1', 'Praha', '11000', 'CZ', '12345678', 'CZ12345678', '',
		'+420111222333', 'test@example.com', 'https://example.com', '/some/logo.png', 1, 1, 'VF', 'test notes', 'cs', ?, ?)`,
		ids.supplierID, now, now)

	// Bank account
	mustExec(t, db, `INSERT INTO bank_accounts (id, supplier_id, name, account_number, iban, swift,
		currency, is_default, qr_type, created_at)
		VALUES (?, ?, 'Hlavni ucet', '123456789/0100', 'CZ1234567890', 'KOMBCZPP', 'CZK', 1, 'spayd', ?)`,
		ids.bankAccountID, ids.supplierID, now)

	// Customer
	mustExec(t, db, `INSERT INTO customers (id, name, street, city, zip, region, country, ico, dic, ic_dph,
		email, phone, default_vat_rate, default_due_days, notes,
		email_custom_template, email_subject_template, email_body_template, created_at, updated_at)
		VALUES (?, 'Test Customer', 'Zakaznicka 2', 'Brno', '60200', 'Jihomoravsky', 'CZ',
		'87654321', 'CZ87654321', '', 'customer@example.com', '+420999888777', 21, 14, 'customer notes',
		0, '', '', ?, ?)`,
		ids.customerID, now, now)

	// Item catalog entry
	mustExec(t, db, `INSERT INTO items (id, description, default_price, default_unit, default_vat_rate,
		category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
		VALUES (?, 'Web development', 1500, 'hod', 21, 'services', 1500, ?, 5, ?, ?)`,
		ids.itemID, ids.customerID, now, now)

	// Invoice
	issueDate := now
	dueDate := now.AddDate(0, 0, 14)
	mustExec(t, db, `INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id,
		status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
		currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
		language, pdf_path, template_id, created_at, updated_at)
		VALUES (?, 'VF26-00001', ?, ?, ?, 'created', ?, ?, NULL, ?, 'bank_transfer', '2600001',
		'CZK', 1.0, 10000, 2100, 12100, 'faktura poznamka', 'internal note',
		'cs', '/path/to/invoice.pdf', 'classic', ?, ?)`,
		ids.invoiceID, ids.supplierID, ids.customerID, ids.bankAccountID,
		issueDate, dueDate, issueDate, now, now)

	// Invoice item
	mustExec(t, db, `INSERT INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
		unit_price, vat_rate, subtotal, vat_amount, total, position)
		VALUES (?, ?, ?, 'Web development', 10, 'hod', 1000, 21, 10000, 2100, 12100, 0)`,
		ids.invoiceItemID, ids.invoiceID, ids.itemID)

	// Customer item
	mustExec(t, db, `INSERT INTO customer_items (id, customer_id, item_id, last_price, last_quantity,
		usage_count, last_used_at)
		VALUES (?, ?, ?, 1500, 10, 3, ?)`,
		ids.customerItemID, ids.customerID, ids.itemID, now)

	// PDF template (custom, non-builtin)
	mustExec(t, db, `INSERT INTO pdf_templates (id, name, template_code, config_json, is_default,
		supplier_id, description, show_logo, show_qr, show_notes, sort_order, is_builtin, yaml_source, parent_id)
		VALUES (?, 'Custom Template', 'custom', '{"color":"blue"}', 0,
		NULL, 'A custom template', 1, 1, 1, 10, 0, 'template: custom', 'classic')`,
		ids.templateID)

	// SMTP config (with password in DB -- export should strip it)
	mustExec(t, db, `INSERT INTO smtp_configs (id, supplier_id, host, port, username,
		password_encrypted, from_name, from_email, use_starttls, enabled, created_at, updated_at)
		VALUES (?, ?, 'smtp.example.com', 587, 'user@example.com',
		'ENCRYPTED_SECRET_PASSWORD', 'Test Sender', 'sender@example.com', 1, 1, ?, ?)`,
		ids.smtpConfigID, ids.supplierID, now, now)

	// Settings
	mustExec(t, db, `INSERT INTO settings (key, value) VALUES ('language', 'cs')`)
	mustExec(t, db, `INSERT INTO settings (key, value) VALUES ('default_due_days', '14')`)

	return ids
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query[:min(80, len(query))], err)
	}
}

func newExportService(db *sql.DB) *ExportService {
	return NewExportService(
		db,
		repository.NewSupplierRepository(db),
		repository.NewBankAccountRepository(db),
		repository.NewCustomerRepository(db),
		repository.NewInvoiceRepository(db),
		repository.NewInvoiceItemRepository(db),
		repository.NewItemRepository(db),
		repository.NewCustomerItemRepository(db),
		repository.NewPDFTemplateRepository(db),
		repository.NewSmtpConfigRepository(db),
		repository.NewSettingsRepository(db),
	)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestExportContainsAllTables verifies that a full export includes all
// expected top-level keys (tables).
func TestExportContainsAllTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	data, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expectedKeys := []string{
		"tidybill_export", "vat_rates", "settings", "suppliers", "bank_accounts",
		"customers", "pdf_templates", "items", "smtp_configs", "customer_items",
		"invoices", "invoice_items",
	}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in export", key)
		}
	}
}

// TestExportStripsSmtpPasswords verifies that SMTP passwords are never
// included in the export.
func TestExportStripsSmtpPasswords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	data, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}

	// The raw JSON must not contain the actual password string.
	if bytes.Contains(data, []byte("ENCRYPTED_SECRET_PASSWORD")) {
		t.Fatal("export contains SMTP password — this is a security issue")
	}

	// Also check the structured data: SmtpConfigExport has no password field.
	var file ExportFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(file.SmtpConfigs) == 0 {
		t.Fatal("expected at least one smtp config in export")
	}
	// Verify no password_encrypted key in JSON
	if bytes.Contains(data, []byte("password_encrypted")) {
		t.Fatal("export contains password_encrypted field")
	}
}

// TestExportStripsPdfPaths verifies that local PDF paths are not in the export.
func TestExportStripsPdfPaths(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	file, err := svc.Export(nil)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Supplier logo_path should be empty.
	for _, s := range file.Suppliers {
		if s.LogoPath != "" {
			t.Errorf("supplier %s has non-empty logo_path: %q", s.ID, s.LogoPath)
		}
	}

	// Invoices should not have pdf_path (InvoiceExport has no PDFPath field at all).
	jsonData, _ := json.Marshal(file)
	if bytes.Contains(jsonData, []byte("pdf_path")) {
		t.Fatal("export contains pdf_path field")
	}
}

// TestRoundTripFullReplace tests: Export -> Import (full_replace) -> Export again
// -> compare JSON output is identical (minus metadata timestamps).
func TestRoundTripFullReplace(t *testing.T) {
	// Source DB with data.
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	srcSvc := newExportService(srcDB)
	exported1, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("first export: %v", err)
	}
	json1, err := json.Marshal(exported1)
	if err != nil {
		t.Fatalf("marshal export 1: %v", err)
	}

	// Destination DB (empty, just migrated).
	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	report, err := importSvc.Import(bytes.NewReader(json1), ImportOptions{Mode: ImportModeFullReplace})
	if err != nil {
		t.Fatalf("import full_replace: %v", err)
	}

	// Verify the report shows some inserts.
	if report.Summary.ToInsert == 0 {
		t.Fatal("expected some inserts in the import report")
	}
	if report.Summary.Conflicts > 0 {
		t.Fatalf("unexpected conflicts: %d", report.Summary.Conflicts)
	}

	// Export from destination.
	dstSvc := newExportService(dstDB)
	exported2, err := dstSvc.Export(nil)
	if err != nil {
		t.Fatalf("second export: %v", err)
	}

	// Compare data (skip metadata which has timestamps).
	compareExportData(t, exported1, exported2)
}

// TestRoundTripSmartMergeSameData tests: Export -> Import (smart_merge with same data)
// -> verify no inserts or updates (all skipped).
func TestRoundTripSmartMergeSameData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	jsonData, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	importSvc := NewImportService(db)
	report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeSmartMerge})
	if err != nil {
		t.Fatalf("smart merge: %v", err)
	}

	// Everything should be skipped since data is identical.
	if report.Summary.ToInsert != 0 {
		t.Errorf("expected 0 inserts, got %d", report.Summary.ToInsert)
	}
	// Settings are always updated (key-merge overwrites), so we only check inserts.
}

// TestRoundTripEncrypted tests: Export (encrypted) -> Import (encrypted)
// -> Export again -> compare.
func TestRoundTripEncrypted(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	srcSvc := newExportService(srcDB)
	passphrase := "test-secure-pass!123"
	encData, err := srcSvc.ExportEncryptedJSON(nil, passphrase)
	if err != nil {
		t.Fatalf("encrypted export: %v", err)
	}

	if !IsEncrypted(encData) {
		t.Fatal("exported data should be encrypted")
	}

	// Import into fresh DB.
	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	report, err := importSvc.Import(bytes.NewReader(encData), ImportOptions{
		Mode:       ImportModeFullReplace,
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("import encrypted: %v", err)
	}
	if report.Summary.ToInsert == 0 {
		t.Fatal("expected some inserts")
	}

	// Export again (unencrypted for comparison).
	dstSvc := newExportService(dstDB)
	exported2, err := dstSvc.Export(nil)
	if err != nil {
		t.Fatalf("second export: %v", err)
	}

	// Also export from source for comparison.
	exported1, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("source re-export: %v", err)
	}

	compareExportData(t, exported1, exported2)
}

// TestExportFiltered verifies that filtering by supplier_ids limits the export
// to only related data.
func TestExportFiltered(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ids := seedTestData(t, db)

	// Add a second supplier with its own data.
	now := time.Now().UTC().Truncate(time.Second)
	mustExec(t, db, `INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic, ic_dph,
		phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language,
		created_at, updated_at)
		VALUES ('sup-other', 'Other Supplier', '', '', '', 'CZ', '', '', '',
		'', '', '', '', 0, 0, 'OT', '', 'cs', ?, ?)`, now, now)
	mustExec(t, db, `INSERT INTO bank_accounts (id, supplier_id, name, account_number, currency, is_default, created_at)
		VALUES ('ba-other', 'sup-other', 'Other Account', '999/0200', 'EUR', 1, ?)`, now)
	mustExec(t, db, `INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id,
		status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
		currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
		language, pdf_path, template_id, created_at, updated_at)
		VALUES ('inv-other', 'OT26-00001', 'sup-other', ?, 'ba-other',
		'created', ?, ?, NULL, ?, 'bank_transfer', '9900001',
		'EUR', 1.0, 5000, 1050, 6050, '', '',
		'cs', '', 'classic', ?, ?)`,
		ids.customerID, now, now.AddDate(0, 0, 14), now, now, now)

	svc := newExportService(db)

	// Filter to only the first supplier.
	filters := &ExportFilters{SupplierIDs: []string{ids.supplierID}}
	file, err := svc.Export(filters)
	if err != nil {
		t.Fatalf("filtered export: %v", err)
	}

	// Should have only the first supplier.
	if len(file.Suppliers) != 1 || file.Suppliers[0].ID != ids.supplierID {
		t.Fatalf("expected 1 supplier (%s), got %d", ids.supplierID, len(file.Suppliers))
	}

	// Bank accounts should only be for the filtered supplier.
	for _, ba := range file.BankAccounts {
		if ba.SupplierID != ids.supplierID {
			t.Errorf("bank account %s belongs to wrong supplier %s", ba.ID, ba.SupplierID)
		}
	}

	// Invoices should only be for the filtered supplier.
	for _, inv := range file.Invoices {
		if inv.SupplierID != ids.supplierID {
			t.Errorf("invoice %s belongs to wrong supplier %s", inv.ID, inv.SupplierID)
		}
	}

	// The "other" invoice should NOT be present.
	for _, inv := range file.Invoices {
		if inv.ID == "inv-other" {
			t.Error("filtered export should not contain inv-other")
		}
	}
}

// TestImportReportCountsMatch verifies that the import report summary
// matches the sum of per-table details.
func TestImportReportCountsMatch(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	svc := newExportService(srcDB)
	jsonData, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeFullReplace})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	// Compute expected totals from details.
	var totalInsert, totalUpdate, totalSkip int
	for _, ts := range report.Details {
		totalInsert += ts.Insert
		totalUpdate += ts.Update
		totalSkip += ts.Skip
	}

	if report.Summary.ToInsert != totalInsert {
		t.Errorf("Summary.ToInsert=%d, but sum of details=%d", report.Summary.ToInsert, totalInsert)
	}
	if report.Summary.ToUpdate != totalUpdate {
		t.Errorf("Summary.ToUpdate=%d, but sum of details=%d", report.Summary.ToUpdate, totalUpdate)
	}
	if report.Summary.ToSkip != totalSkip {
		t.Errorf("Summary.ToSkip=%d, but sum of details=%d", report.Summary.ToSkip, totalSkip)
	}
	if report.Summary.Conflicts != len(report.Conflicts) {
		t.Errorf("Summary.Conflicts=%d, but len(Conflicts)=%d", report.Summary.Conflicts, len(report.Conflicts))
	}
	if report.Summary.Warnings != len(report.Warnings) {
		t.Errorf("Summary.Warnings=%d, but len(Warnings)=%d", report.Summary.Warnings, len(report.Warnings))
	}
}

// TestPreviewMode verifies that preview mode does not modify data.
func TestPreviewMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newExportService(db)
	// Export empty DB.
	emptyExport, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export empty: %v", err)
	}

	// Seed data and export.
	seedTestData(t, db)
	fullExport, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export full: %v", err)
	}

	// Import in preview mode against a fresh DB -- should not change anything.
	freshDB := setupTestDB(t)
	defer freshDB.Close()

	importSvc := NewImportService(freshDB)
	report, err := importSvc.Import(bytes.NewReader(fullExport), ImportOptions{Mode: ImportModePreview})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}

	// Preview should show inserts.
	if report.Summary.ToInsert == 0 {
		t.Error("preview should report pending inserts")
	}

	// Verify DB is still empty (preview is read-only).
	freshSvc := newExportService(freshDB)
	afterPreview, err := freshSvc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export after preview: %v", err)
	}

	// The fresh DB export should match the empty export (no data inserted).
	var emptyFile, afterFile ExportFile
	json.Unmarshal(emptyExport, &emptyFile)
	json.Unmarshal(afterPreview, &afterFile)

	if len(afterFile.Suppliers) != len(emptyFile.Suppliers) {
		t.Errorf("preview modified suppliers: before=%d, after=%d",
			len(emptyFile.Suppliers), len(afterFile.Suppliers))
	}
	if len(afterFile.Invoices) != len(emptyFile.Invoices) {
		t.Errorf("preview modified invoices: before=%d, after=%d",
			len(emptyFile.Invoices), len(afterFile.Invoices))
	}
}

// TestForceImport verifies the force import mode overwrites existing data.
func TestForceImport(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	svc := newExportService(srcDB)
	jsonData, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Import into fresh DB with force mode.
	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeForce})
	if err != nil {
		t.Fatalf("force import: %v", err)
	}

	if report.Summary.ToInsert == 0 {
		t.Error("expected inserts on fresh DB")
	}

	// Now import again with force -- should update existing.
	report2, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeForce})
	if err != nil {
		t.Fatalf("second force import: %v", err)
	}

	// On second import, most things should be updates.
	if report2.Summary.ToUpdate == 0 {
		t.Error("expected updates on second force import")
	}
}

// TestExportImportExportIdentical does a full cycle:
// Export -> Import (full_replace) -> Export -> compare JSON output.
func TestExportImportExportIdentical(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	srcSvc := newExportService(srcDB)
	export1, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("export 1: %v", err)
	}
	json1, _ := json.Marshal(export1)

	// Import into fresh DB.
	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	_, err = importSvc.Import(bytes.NewReader(json1), ImportOptions{Mode: ImportModeFullReplace})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	// Export from destination.
	dstSvc := newExportService(dstDB)
	export2, err := dstSvc.Export(nil)
	if err != nil {
		t.Fatalf("export 2: %v", err)
	}

	// Re-import into a third DB and export again.
	thirdDB := setupTestDB(t)
	defer thirdDB.Close()

	json2, _ := json.Marshal(export2)
	importSvc2 := NewImportService(thirdDB)
	_, err = importSvc2.Import(bytes.NewReader(json2), ImportOptions{Mode: ImportModeFullReplace})
	if err != nil {
		t.Fatalf("import 2: %v", err)
	}

	thirdSvc := newExportService(thirdDB)
	export3, err := thirdSvc.Export(nil)
	if err != nil {
		t.Fatalf("export 3: %v", err)
	}

	// export2 and export3 should be identical (proving stability).
	compareExportData(t, export2, export3)
}

// TestExportAllTablesPresent verifies that all expected tables are represented
// in the export even when some may be empty.
func TestExportAllTablesPresent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	// Do NOT seed data -- export an empty-ish DB.

	svc := newExportService(db)
	file, err := svc.Export(nil)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// These should be non-nil slices (may be empty but the key should exist in JSON).
	data, _ := json.Marshal(file)
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)

	tables := []string{
		"suppliers", "bank_accounts", "customers", "invoices", "invoice_items",
		"items", "customer_items", "pdf_templates", "settings", "vat_rates",
		"smtp_configs",
	}
	for _, tbl := range tables {
		if _, ok := raw[tbl]; !ok {
			t.Errorf("table %q missing from export JSON", tbl)
		}
	}
}

// TestImportWrongPassphrase verifies encrypted import fails with wrong passphrase.
func TestImportWrongPassphrase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	encData, err := svc.ExportEncryptedJSON(nil, "correct-pass!123")
	if err != nil {
		t.Fatalf("encrypt export: %v", err)
	}

	freshDB := setupTestDB(t)
	defer freshDB.Close()

	importSvc := NewImportService(freshDB)
	_, err = importSvc.Import(bytes.NewReader(encData), ImportOptions{
		Mode:       ImportModeFullReplace,
		Passphrase: "wrong-pass!456",
	})
	if err == nil {
		t.Fatal("expected error with wrong passphrase")
	}
}

// TestImportEncryptedNoPassphrase verifies encrypted import fails without passphrase.
func TestImportEncryptedNoPassphrase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	encData, err := svc.ExportEncryptedJSON(nil, "test-pass!123")
	if err != nil {
		t.Fatalf("encrypt export: %v", err)
	}

	freshDB := setupTestDB(t)
	defer freshDB.Close()

	importSvc := NewImportService(freshDB)
	_, err = importSvc.Import(bytes.NewReader(encData), ImportOptions{
		Mode: ImportModeFullReplace,
	})
	if err == nil {
		t.Fatal("expected error when no passphrase provided for encrypted file")
	}
	if !strings.Contains(err.Error(), "passphrase required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestExportExcludeSettings verifies that exclude_settings filter works.
func TestExportExcludeSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	file, err := svc.Export(&ExportFilters{ExcludeSettings: true})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(file.Settings) != 0 {
		t.Errorf("expected 0 settings with exclude_settings=true, got %d", len(file.Settings))
	}
	// Other data should still be present.
	if len(file.Suppliers) == 0 {
		t.Error("suppliers should still be present when settings are excluded")
	}
}

// TestSmartMergeNewData verifies that smart merge inserts new data.
func TestSmartMergeNewData(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	svc := newExportService(srcDB)
	jsonData, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Import into fresh DB with smart merge.
	dstDB := setupTestDB(t)
	defer dstDB.Close()

	importSvc := NewImportService(dstDB)
	report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeSmartMerge})
	if err != nil {
		t.Fatalf("smart merge: %v", err)
	}

	if report.Summary.ToInsert == 0 {
		t.Error("expected inserts when merging into empty DB")
	}

	// Verify data is actually present.
	var count int
	dstDB.QueryRow("SELECT COUNT(*) FROM suppliers").Scan(&count)
	if count == 0 {
		t.Error("no suppliers after smart merge")
	}
	dstDB.QueryRow("SELECT COUNT(*) FROM invoices").Scan(&count)
	if count == 0 {
		t.Error("no invoices after smart merge")
	}
}

// TestImportReportHasAllTables verifies that the import report contains
// entries for all tables regardless of import mode.
func TestImportReportHasAllTables(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)

	svc := newExportService(srcDB)
	jsonData, err := svc.ExportJSON(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	modes := []string{ImportModeFullReplace, ImportModeSmartMerge, ImportModeForce}
	expectedTables := []string{
		"vat_rates", "settings", "suppliers", "bank_accounts", "customers",
		"pdf_templates", "items", "smtp_configs", "customer_items",
		"invoices", "invoice_items",
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			dstDB := setupTestDB(t)
			defer dstDB.Close()

			importSvc := NewImportService(dstDB)
			report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: mode})
			if err != nil {
				t.Fatalf("import (%s): %v", mode, err)
			}

			for _, tbl := range expectedTables {
				if _, ok := report.Details[tbl]; !ok {
					t.Errorf("mode %s: missing table %q in import report details", mode, tbl)
				}
			}
		})
	}
}

// TestExportMetadata verifies that export metadata is populated correctly.
func TestExportMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	file, err := svc.Export(nil)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if file.Metadata.FormatVersion != FormatVersion {
		t.Errorf("format_version=%d, want %d", file.Metadata.FormatVersion, FormatVersion)
	}
	if file.Metadata.ExportMode != "full" {
		t.Errorf("export_mode=%q, want %q", file.Metadata.ExportMode, "full")
	}
	if file.Metadata.ExportedAt.IsZero() {
		t.Error("exported_at should not be zero")
	}
	if file.Metadata.SchemaVersion == 0 {
		t.Error("schema_version should not be 0")
	}
}

// TestFilteredExportMetadata verifies that filtered export has correct metadata.
func TestFilteredExportMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	svc := newExportService(db)
	filters := &ExportFilters{SupplierIDs: []string{"sup-test-001"}}
	file, err := svc.Export(filters)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if file.Metadata.ExportMode != "filtered" {
		t.Errorf("export_mode=%q, want %q", file.Metadata.ExportMode, "filtered")
	}
	if file.Metadata.Filters == nil {
		t.Fatal("filters should not be nil for filtered export")
	}
}

// TestInvoiceNumberCollisionOnMerge tests that importing an invoice with
// a conflicting invoice_number is handled correctly.
func TestInvoiceNumberCollisionOnMerge(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestData(t, db)

	// Export the current data.
	svc := newExportService(db)
	file, err := svc.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Modify the export: change invoice ID but keep same invoice_number.
	if len(file.Invoices) == 0 {
		t.Fatal("no invoices in export")
	}
	file.Invoices[0].ID = "inv-collision-new"
	// Keep same InvoiceNumber and SupplierID.

	jsonData, _ := json.Marshal(file)

	importSvc := NewImportService(db)
	report, err := importSvc.Import(bytes.NewReader(jsonData), ImportOptions{Mode: ImportModeSmartMerge})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	if report.Summary.Conflicts == 0 {
		t.Error("expected invoice number collision conflict")
	}

	// Verify the conflict is reported.
	foundCollision := false
	for _, c := range report.Conflicts {
		if c.Type == "invoice_number_collision" {
			foundCollision = true
			break
		}
	}
	if !foundCollision {
		t.Error("expected an invoice_number_collision conflict in report")
	}
}

// ---------------------------------------------------------------------------
// Comparison helpers
// ---------------------------------------------------------------------------

// compareExportData compares two ExportFile structures, ignoring metadata
// (timestamps, device_id, etc.) that naturally differ between exports.
func compareExportData(t *testing.T, a, b *ExportFile) {
	t.Helper()

	// Compare suppliers.
	if len(a.Suppliers) != len(b.Suppliers) {
		t.Errorf("suppliers count: %d vs %d", len(a.Suppliers), len(b.Suppliers))
	} else {
		sortSuppliers(a.Suppliers)
		sortSuppliers(b.Suppliers)
		for i := range a.Suppliers {
			if a.Suppliers[i].ID != b.Suppliers[i].ID {
				t.Errorf("supplier[%d] ID: %s vs %s", i, a.Suppliers[i].ID, b.Suppliers[i].ID)
			}
			if a.Suppliers[i].Name != b.Suppliers[i].Name {
				t.Errorf("supplier[%d] Name: %s vs %s", i, a.Suppliers[i].Name, b.Suppliers[i].Name)
			}
		}
	}

	// Compare bank accounts.
	if len(a.BankAccounts) != len(b.BankAccounts) {
		t.Errorf("bank_accounts count: %d vs %d", len(a.BankAccounts), len(b.BankAccounts))
	}

	// Compare customers.
	if len(a.Customers) != len(b.Customers) {
		t.Errorf("customers count: %d vs %d", len(a.Customers), len(b.Customers))
	}

	// Compare invoices.
	if len(a.Invoices) != len(b.Invoices) {
		t.Errorf("invoices count: %d vs %d", len(a.Invoices), len(b.Invoices))
	} else {
		sortInvoices(a.Invoices)
		sortInvoices(b.Invoices)
		for i := range a.Invoices {
			if a.Invoices[i].ID != b.Invoices[i].ID {
				t.Errorf("invoice[%d] ID: %s vs %s", i, a.Invoices[i].ID, b.Invoices[i].ID)
			}
			if a.Invoices[i].InvoiceNumber != b.Invoices[i].InvoiceNumber {
				t.Errorf("invoice[%d] Number: %s vs %s", i, a.Invoices[i].InvoiceNumber, b.Invoices[i].InvoiceNumber)
			}
			if a.Invoices[i].Total != b.Invoices[i].Total {
				t.Errorf("invoice[%d] Total: %f vs %f", i, a.Invoices[i].Total, b.Invoices[i].Total)
			}
		}
	}

	// Compare invoice items.
	if len(a.InvoiceItems) != len(b.InvoiceItems) {
		t.Errorf("invoice_items count: %d vs %d", len(a.InvoiceItems), len(b.InvoiceItems))
	}

	// Compare items.
	if len(a.Items) != len(b.Items) {
		t.Errorf("items count: %d vs %d", len(a.Items), len(b.Items))
	}

	// Compare customer items.
	if len(a.CustomerItems) != len(b.CustomerItems) {
		t.Errorf("customer_items count: %d vs %d", len(a.CustomerItems), len(b.CustomerItems))
	}

	// Compare pdf templates.
	if len(a.PDFTemplates) != len(b.PDFTemplates) {
		t.Errorf("pdf_templates count: %d vs %d", len(a.PDFTemplates), len(b.PDFTemplates))
	}

	// Compare settings.
	if len(a.Settings) != len(b.Settings) {
		t.Errorf("settings count: %d vs %d", len(a.Settings), len(b.Settings))
	}

	// Compare vat rates.
	if len(a.VatRates) != len(b.VatRates) {
		t.Errorf("vat_rates count: %d vs %d", len(a.VatRates), len(b.VatRates))
	}

	// Compare smtp configs.
	if len(a.SmtpConfigs) != len(b.SmtpConfigs) {
		t.Errorf("smtp_configs count: %d vs %d", len(a.SmtpConfigs), len(b.SmtpConfigs))
	}
}

func sortSuppliers(s []model.Supplier) {
	sort.Slice(s, func(i, j int) bool { return s[i].ID < s[j].ID })
}

func sortInvoices(inv []InvoiceExport) {
	sort.Slice(inv, func(i, j int) bool { return inv[i].ID < inv[j].ID })
}
