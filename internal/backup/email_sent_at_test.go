package backup

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/model"
)

// TestEmailSentAtRoundTrip is the regression test for a real bug shipped on
// Android: importing a backup that contained an invoice with email_sent_at
// set succeeded silently, then `GET /api/invoices` returned
//
//	"sql: Scan error on column index 19, name \"email_sent_at\":
//	 unsupported Scan, storing driver.Value type string into type *time.Time"
//
// because modernc.org/sqlite's parseTime couldn't recognize the format the
// import wrote. Pre-existing seedTestData omitted email_sent_at, so the bug
// never surfaced in the standard roundtrip tests.
func TestEmailSentAtRoundTrip(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	ids := seedTestData(t, srcDB)

	mustExec(t, srcDB, `UPDATE invoices SET email_sent_at = datetime('now') WHERE id = ?`, ids.invoiceID)

	srcSvc := newExportService(srcDB)
	exported, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	jsonBytes, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)
	if _, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
		t.Fatalf("import: %v", err)
	}

	repo := repository.NewInvoiceRepository(dstDB)
	invoices, err := repo.List("", "")
	if err != nil {
		t.Fatalf("List FAILED (this is the bug): %v", err)
	}
	if len(invoices) != 1 || invoices[0].EmailSentAt == nil {
		t.Fatalf("expected 1 invoice with non-nil EmailSentAt, got %d invoices, EmailSentAt=%v",
			len(invoices), invoices[0].EmailSentAt)
	}
}

// TestRoundTripAllNullableTimesPopulated covers every nullable time field
// across every table — paid_date, email_sent_at, etc. — to guard against
// the same class of bug regressing on a different field.
func TestRoundTripAllNullableTimesPopulated(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	ids := seedTestData(t, srcDB)

	mustExec(t, srcDB, `UPDATE invoices SET email_sent_at = datetime('now', '-2 days'),
		paid_date = datetime('now', '-1 day') WHERE id = ?`, ids.invoiceID)

	srcSvc := newExportService(srcDB)
	exported, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	jsonBytes, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)
	if _, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Read through every public listing endpoint that touches a time column.
	repo := repository.NewInvoiceRepository(dstDB)
	if _, err := repo.List("", ""); err != nil {
		t.Errorf("InvoiceRepository.List: %v", err)
	}
	if _, err := repo.ListUnpaid(); err != nil {
		t.Errorf("InvoiceRepository.ListUnpaid: %v", err)
	}
	if _, err := repo.ListFiltered("", "", time.Time{}, time.Time{}); err != nil {
		t.Errorf("InvoiceRepository.ListFiltered: %v", err)
	}
	if _, err := repository.NewCustomerRepository(dstDB).List(); err != nil {
		t.Errorf("CustomerRepository.List: %v", err)
	}
	if _, err := repository.NewSupplierRepository(dstDB).List(); err != nil {
		t.Errorf("SupplierRepository.List: %v", err)
	}
}

// TestImportBackwardCompatMissingFields simulates an older backup format
// that doesn't contain newer columns (e.g. a backup made before
// email_sent_at, customer email templates, retention settings existed).
// JSON unmarshal must produce zero-values, and import must succeed without
// referencing fields the backup doesn't carry.
func TestImportBackwardCompatMissingFields(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	seedTestData(t, srcDB)
	srcSvc := newExportService(srcDB)
	exported, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	full, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Strip a handful of newer-looking fields from the JSON to mimic an
	// older export. Whatever the parser does with missing fields, it
	// should not crash.
	stripped := string(full)
	for _, field := range []string{
		`"email_sent_at":`,
		`"email_custom_template":`,
		`"email_subject_template":`,
		`"email_body_template":`,
		`"is_default":`, // covers a reasonable older surface
	} {
		// Crude but sufficient: blank out the value so JSON stays valid via field absence.
		// Real "missing field" simulation: rewrite to remove the key entirely.
		stripped = stripFieldFromJSON(stripped, field)
	}

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)
	if _, err := importSvc.Import(bytes.NewReader([]byte(stripped)), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
		t.Fatalf("import after stripping fields: %v", err)
	}

	// Reads must still work end-to-end.
	repo := repository.NewInvoiceRepository(dstDB)
	if _, err := repo.List("", ""); err != nil {
		t.Errorf("List after stripped-field import: %v", err)
	}
}

// TestRoundTripDifferentTimezones checks that times stored in non-UTC zones
// round-trip without losing the moment-in-time (whether or not the zone
// label is preserved).
func TestRoundTripDifferentTimezones(t *testing.T) {
	zones := []string{"UTC", "Europe/Prague", "America/Los_Angeles", "Asia/Tokyo"}
	for _, zone := range zones {
		t.Run(zone, func(t *testing.T) {
			loc, err := time.LoadLocation(zone)
			if err != nil {
				t.Skipf("zone %q not available: %v", zone, err)
			}
			srcDB := setupTestDB(t)
			defer srcDB.Close()
			ids := seedTestData(t, srcDB)
			when := time.Date(2026, 4, 28, 12, 0, 0, 0, loc)
			mustExec(t, srcDB, `UPDATE invoices SET email_sent_at = ? WHERE id = ?`, when, ids.invoiceID)

			srcSvc := newExportService(srcDB)
			exported, _ := srcSvc.Export(nil)
			jsonBytes, _ := json.Marshal(exported)

			dstDB := setupTestDB(t)
			defer dstDB.Close()
			importSvc := NewImportService(dstDB)
			if _, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
				t.Fatalf("import (%s): %v", zone, err)
			}

			invoices, err := repository.NewInvoiceRepository(dstDB).List("", "")
			if err != nil {
				t.Fatalf("List (%s): %v", zone, err)
			}
			if len(invoices) != 1 || invoices[0].EmailSentAt == nil {
				t.Fatalf("zone=%s: expected 1 invoice with EmailSentAt set", zone)
			}
			if !invoices[0].EmailSentAt.Equal(when) {
				t.Errorf("zone=%s: time drift after roundtrip — got %v, want %v",
					zone, invoices[0].EmailSentAt, when)
			}
		})
	}
}

// TestEmptyExportImport: zero-row tables across the backup. Should
// produce a clean empty target DB without any FK or NULL constraint errors.
func TestEmptyExportImport(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	srcSvc := newExportService(srcDB)
	exported, err := srcSvc.Export(nil)
	if err != nil {
		t.Fatalf("empty export: %v", err)
	}
	if len(exported.Invoices) != 0 {
		t.Fatalf("expected zero invoices in fresh DB, got %d", len(exported.Invoices))
	}
	jsonBytes, _ := json.Marshal(exported)

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)
	if _, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
		t.Fatalf("import empty: %v", err)
	}

	invoices, err := repository.NewInvoiceRepository(dstDB).List("", "")
	if err != nil {
		t.Fatalf("List empty DB: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("expected empty list, got %d", len(invoices))
	}
}

// TestRoundTripUnicodeAndSpecialChars: Czech accents, emojis, and quote
// characters in user-provided strings (notes, customer names, etc.) must
// survive JSON marshal/unmarshal and SQLite TEXT roundtrip.
func TestRoundTripUnicodeAndSpecialChars(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	ids := seedTestData(t, srcDB)

	const tricky = "Příliš žluťoučký kůň 🐎 \"quoted\" 'apostrophe' \\backslash\n\tnewline"
	mustExec(t, srcDB, `UPDATE invoices SET notes = ? WHERE id = ?`, tricky, ids.invoiceID)
	mustExec(t, srcDB, `UPDATE customers SET name = ? WHERE id = ?`, tricky, ids.customerID)

	srcSvc := newExportService(srcDB)
	exported, _ := srcSvc.Export(nil)
	jsonBytes, _ := json.Marshal(exported)

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)
	if _, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace}); err != nil {
		t.Fatalf("import: %v", err)
	}

	invoices, err := repository.NewInvoiceRepository(dstDB).List("", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(invoices) != 1 || invoices[0].Notes != tricky {
		t.Errorf("notes drift after roundtrip:\n  want %q\n  got  %q", tricky,
			func() string {
				if len(invoices) > 0 {
					return invoices[0].Notes
				}
				return "<no invoices>"
			}())
	}
}

// TestImportPreviewVsFullReplaceCounts: preview report should report the
// same insert count as the full_replace actually performs. Catches
// "preview says 70 will be updated, then 0 are imported" UX bug.
func TestImportPreviewVsFullReplaceCounts(t *testing.T) {
	srcDB := setupTestDB(t)
	defer srcDB.Close()
	ids := seedTestData(t, srcDB)
	mustExec(t, srcDB, `UPDATE invoices SET email_sent_at = datetime('now') WHERE id = ?`, ids.invoiceID)

	srcSvc := newExportService(srcDB)
	exported, _ := srcSvc.Export(nil)
	jsonBytes, _ := json.Marshal(exported)

	dstDB := setupTestDB(t)
	defer dstDB.Close()
	importSvc := NewImportService(dstDB)

	// Preview defaults to smart-merge semantics; force it to mirror
	// full-replace counting.
	previewRep, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{
		Mode:        ImportModePreview,
		PreviewMode: "replace",
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	previewInserts := previewRep.Summary.ToInsert

	fullRep, err := importSvc.Import(bytes.NewReader(jsonBytes), ImportOptions{Mode: ImportModeFullReplace})
	if err != nil {
		t.Fatalf("full_replace: %v", err)
	}
	fullInserts := fullRep.Summary.ToInsert

	if previewInserts == 0 {
		t.Fatal("preview report claimed 0 inserts; should be > 0")
	}
	if previewInserts != fullInserts {
		t.Errorf("preview/actual insert mismatch — preview=%d actual=%d", previewInserts, fullInserts)
	}

	// And the data must actually be visible end-to-end.
	invoices, err := repository.NewInvoiceRepository(dstDB).List("", "")
	if err != nil {
		t.Fatalf("List after full_replace: %v", err)
	}
	if len(invoices) == 0 {
		t.Fatal("post-import invoice list is empty even though report claimed inserts")
	}
}

// stripFieldFromJSON removes occurrences of a JSON object key (and its
// value) from a string. Crude but enough for backward-compat simulation
// in tests. Doesn't handle nested objects in the value position.
func stripFieldFromJSON(s, key string) string {
	for {
		i := strings.Index(s, key)
		if i < 0 {
			return s
		}
		// find the comma terminator or closing brace
		j := i + len(key)
		depth := 0
		for j < len(s) {
			c := s[j]
			if c == '{' || c == '[' {
				depth++
			}
			if c == '}' || c == ']' {
				if depth == 0 {
					break
				}
				depth--
			}
			if c == ',' && depth == 0 {
				j++
				break
			}
			j++
		}
		// Strip the key:value pair plus trailing comma if present.
		s = s[:i] + s[j:]
	}
}

// _ keeps the model import alive even if direct uses are removed by edits.
var _ = model.Invoice{}
