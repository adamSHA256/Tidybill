package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

func (c *CLI) syncMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.sync"))
		fmt.Println()

		fmt.Printf("  1) %s\n", i18n.T("sync.export_all"))
		fmt.Printf("  2) %s\n", i18n.T("sync.export_filtered"))
		fmt.Printf("  3) %s\n", i18n.T("sync.import_merge"))
		fmt.Printf("  4) %s\n", i18n.T("sync.import_replace"))
		fmt.Printf("  5) %s\n", i18n.T("sync.import_force"))
		fmt.Printf("  6) %s\n", i18n.T("sync.import_preview"))
		fmt.Printf("  %s\n", i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch choice {
		case "1":
			c.syncExportAll()
		case "2":
			c.syncExportFiltered()
		case "3":
			c.syncImport(backup.ImportModeSmartMerge)
		case "4":
			c.syncImport(backup.ImportModeFullReplace)
		case "5":
			c.syncImport(backup.ImportModeForce)
		case "6":
			c.syncImport(backup.ImportModePreview)
		case "0", "q":
			return
		}
	}
}

// ── Export ──────────────────────────────────────────────────────────────────

func (c *CLI) syncExportAll() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n\n", i18n.T("sync.export_all"))

	defaultName := fmt.Sprintf("tidybill-backup-%s.tidybill", time.Now().Format("2006-01-02"))
	path := c.promptDefault(i18n.T("sync.prompt_output_file"), defaultName)
	if path == "" {
		return
	}

	c.doExport(path, nil)
}

func (c *CLI) syncExportFiltered() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n\n", i18n.T("sync.export_filtered"))

	filters := &backup.ExportFilters{}

	// Filter by supplier
	if c.confirm(i18n.T("sync.filter_by_supplier")) {
		suppliers, err := c.suppliers.List()
		if err != nil || len(suppliers) == 0 {
			c.printError(i18n.T("sync.no_suppliers"))
			c.waitEnter()
			return
		}
		fmt.Println()
		for i, s := range suppliers {
			fmt.Printf("  %d) %s\n", i+1, s.Name)
		}
		fmt.Println()
		input := c.prompt(i18n.T("sync.pick_suppliers"))
		if input != "" {
			parts := strings.Split(input, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				idx := 0
				fmt.Sscanf(p, "%d", &idx)
				idx--
				if idx >= 0 && idx < len(suppliers) {
					filters.SupplierIDs = append(filters.SupplierIDs, suppliers[idx].ID)
				}
			}
		}
	}

	// Skip paid invoices older than N years
	fmt.Println()
	if c.confirm(i18n.T("sync.filter_skip_old_paid")) {
		years := c.promptInt(i18n.T("sync.skip_years"), 2)
		if years > 0 {
			filters.SkipPaidOlderThanYears = &years
		}
	}

	// Exclude settings
	fmt.Println()
	if c.confirm(i18n.T("sync.filter_exclude_settings")) {
		filters.ExcludeSettings = true
	}

	fmt.Println()
	defaultName := fmt.Sprintf("tidybill-backup-%s.tidybill", time.Now().Format("2006-01-02"))
	path := c.promptDefault(i18n.T("sync.prompt_output_file"), defaultName)
	if path == "" {
		return
	}

	c.doExport(path, filters)
}

func (c *CLI) doExport(path string, filters *backup.ExportFilters) {
	smtpConfigs := repository.NewSmtpConfigRepository(c.db.DB)
	svc := backup.NewExportService(
		c.db.DB, c.suppliers, c.bankAccs, c.customers,
		c.invoices, c.invItems, c.items, c.custItems,
		c.templates, smtpConfigs, c.settings,
	)

	// Ask about encryption.
	fmt.Println()
	var passphrase string
	if c.confirm("Encrypt? (y/n)") {
		for {
			pass1 := c.prompt("Passphrase: ")
			if len(pass1) < 8 {
				c.printError("Passphrase must be at least 8 characters")
				continue
			}
			pass2 := c.prompt("Confirm passphrase: ")
			if pass1 != pass2 {
				c.printError("Passphrases do not match")
				continue
			}
			passphrase = pass1
			break
		}
	}

	fmt.Println()
	fmt.Println(i18n.T("sync.exporting"))

	var data []byte
	var err error
	if passphrase != "" {
		data, err = svc.ExportEncryptedJSON(filters, passphrase)
	} else {
		data, err = svc.ExportJSON(filters)
	}
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// For encrypted exports, show summary from ExportJSON (re-export unencrypted for counts).
	// For unencrypted exports, parse back to get entity counts.
	var file backup.ExportFile
	if passphrase != "" {
		plainData, plainErr := svc.ExportJSON(filters)
		if plainErr == nil {
			_ = json.Unmarshal(plainData, &file)
		}
	} else {
		_ = json.Unmarshal(data, &file)
	}

	fmt.Println()
	c.printSuccess(i18n.Tf("sync.export_done", path))
	if passphrase != "" {
		fmt.Println("  (encrypted)")
	}
	fmt.Println()
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_suppliers"), len(file.Suppliers))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_customers"), len(file.Customers))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_invoices"), len(file.Invoices))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_items"), len(file.Items))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_templates"), len(file.PDFTemplates))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_settings"), len(file.Settings))
	fmt.Printf("  %-20s %s\n", i18n.T("sync.file_size"), formatFileSize(int64(len(data))))
	c.waitEnter()
}

// ── Import ─────────────────────────────────────────────────────────────────

func (c *CLI) syncImport(mode string) {
	c.clearScreen()

	modeLabels := map[string]string{
		backup.ImportModeSmartMerge:  i18n.T("sync.import_merge"),
		backup.ImportModeFullReplace: i18n.T("sync.import_replace"),
		backup.ImportModeForce:       i18n.T("sync.import_force"),
		backup.ImportModePreview:     i18n.T("sync.import_preview"),
	}
	fmt.Printf("=== %s ===\n\n", modeLabels[mode])

	path, goBack := c.promptWithBack(i18n.T("sync.prompt_input_file"))
	if goBack || path == "" {
		return
	}

	// Read the entire file to check for encryption.
	rawData, err := os.ReadFile(path)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Detect encryption and prompt for passphrase if needed.
	var passphrase string
	var jsonData []byte
	if backup.IsEncrypted(rawData) {
		fmt.Println()
		fmt.Println("  File is encrypted")
		passphrase = c.prompt("Passphrase: ")
		if passphrase == "" {
			c.printError("Passphrase is required for encrypted files")
			c.waitEnter()
			return
		}
		decrypted, decErr := backup.DecryptExport(rawData, passphrase)
		if decErr != nil {
			c.printError(decErr.Error())
			c.waitEnter()
			return
		}
		jsonData = decrypted
	} else {
		jsonData = rawData
	}

	// Decode just enough to show metadata.
	var file backup.ExportFile
	if err := json.Unmarshal(jsonData, &file); err != nil {
		c.printError(i18n.Tf("sync.invalid_file", err))
		c.waitEnter()
		return
	}

	// Show metadata.
	fmt.Println()
	fmt.Printf("  %-20s %s\n", i18n.T("sync.meta_exported_at"), file.Metadata.ExportedAt.Local().Format("02.01.2006 15:04"))
	fmt.Printf("  %-20s %s\n", i18n.T("sync.meta_device_id"), file.Metadata.DeviceID)
	fmt.Printf("  %-20s %d\n", i18n.T("sync.meta_format_ver"), file.Metadata.FormatVersion)
	fmt.Printf("  %-20s %s\n", i18n.T("sync.meta_app_ver"), file.Metadata.AppVersion)
	fmt.Printf("  %-20s %s\n", i18n.T("sync.meta_mode"), file.Metadata.ExportMode)
	fmt.Println()

	// Show entity counts.
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_suppliers"), len(file.Suppliers))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_customers"), len(file.Customers))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_invoices"), len(file.Invoices))
	fmt.Printf("  %-20s %d\n", i18n.T("sync.count_items"), len(file.Items))
	fmt.Println()

	// Confirm for destructive modes.
	if mode == backup.ImportModeFullReplace {
		fmt.Println(i18n.T("sync.warn_replace"))
		fmt.Println()
		if !c.confirm(i18n.T("sync.confirm_replace")) {
			return
		}
	} else if mode == backup.ImportModeForce {
		fmt.Println(i18n.T("sync.warn_force"))
		fmt.Println()
		if !c.confirm(i18n.T("sync.confirm_force")) {
			return
		}
	}

	svc := backup.NewImportService(c.db.DB)
	opts := backup.ImportOptions{
		Mode:                  mode,
		Passphrase:            passphrase,
		InvoiceNumberConflict: "skip",
	}

	fmt.Println()
	fmt.Println(i18n.T("sync.importing"))

	// Import from the raw data (Import will handle decryption via passphrase if set).
	report, err := svc.Import(bytes.NewReader(rawData), opts)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printImportReport(report, mode == backup.ImportModePreview)
	c.waitEnter()
}

// printImportReport displays the import results as a table.
func (c *CLI) printImportReport(report *backup.ImportReport, isPreview bool) {
	tables := []struct {
		label string
		key   string
	}{
		{i18n.T("sync.table_suppliers"), "suppliers"},
		{i18n.T("sync.table_bank_accounts"), "bank_accounts"},
		{i18n.T("sync.table_customers"), "customers"},
		{i18n.T("sync.table_items"), "items"},
		{i18n.T("sync.table_invoices"), "invoices"},
		{i18n.T("sync.table_invoice_items"), "invoice_items"},
		{i18n.T("sync.table_settings"), "settings"},
		{i18n.T("sync.table_vat_rates"), "vat_rates"},
		{i18n.T("sync.table_templates"), "pdf_templates"},
		{i18n.T("sync.table_smtp"), "smtp_configs"},
		{i18n.T("sync.table_cust_items"), "customer_items"},
	}

	type row struct {
		label                  string
		insert, update, skip   int
	}
	var rows []row
	maxLabel := 10
	for _, t := range tables {
		s, ok := report.Details[t.key]
		if !ok || (s.Insert == 0 && s.Update == 0 && s.Skip == 0) {
			continue
		}
		if len([]rune(t.label)) > maxLabel {
			maxLabel = len([]rune(t.label))
		}
		rows = append(rows, row{t.label, s.Insert, s.Update, s.Skip})
	}

	fmt.Println()
	if len(rows) > 0 {
		hdr := i18n.T("sync.col_table")
		insHdr := i18n.T("sync.col_inserted")
		updHdr := i18n.T("sync.col_updated")
		skipHdr := i18n.T("sync.col_skipped")

		pad := func(n int) string {
			return strings.Repeat("\u2500", n)
		}

		hdrFmt := fmt.Sprintf("  \u250c\u2500%%-%ds\u2500\u252c\u2500%%8s\u2500\u252c\u2500%%7s\u2500\u252c\u2500%%7s\u2500\u2510\n", maxLabel)
		lblFmt := fmt.Sprintf("  \u2502 %%-%ds \u2502 %%8s \u2502 %%7s \u2502 %%7s \u2502\n", maxLabel)
		sepFmt := fmt.Sprintf("  \u251c\u2500%%-%ds\u2500\u253c\u2500%%8s\u2500\u253c\u2500%%7s\u2500\u253c\u2500%%7s\u2500\u2524\n", maxLabel)
		rowFmt := fmt.Sprintf("  \u2502 %%-%ds \u2502 %%8d \u2502 %%7d \u2502 %%7d \u2502\n", maxLabel)
		botFmt := fmt.Sprintf("  \u2514\u2500%%-%ds\u2500\u2534\u2500%%8s\u2500\u2534\u2500%%7s\u2500\u2534\u2500%%7s\u2500\u2518\n", maxLabel)

		fmt.Printf(hdrFmt, pad(maxLabel), pad(8), pad(7), pad(7))
		fmt.Printf(lblFmt, hdr, insHdr, updHdr, skipHdr)
		fmt.Printf(sepFmt, pad(maxLabel), pad(8), pad(7), pad(7))
		for _, r := range rows {
			fmt.Printf(rowFmt, r.label, r.insert, r.update, r.skip)
		}
		fmt.Printf(botFmt, pad(maxLabel), pad(8), pad(7), pad(7))
	}

	for _, cf := range report.Conflicts {
		fmt.Printf("  \u26a0 %s: %s %s (%s)\n", i18n.T("sync.conflict"), cf.Table, cf.Description, cf.Resolution)
	}
	for _, w := range report.Warnings {
		fmt.Printf("  \u26a0 %s: %s %s (%s)\n", i18n.T("sync.warning"), w.Table, w.Description, w.Resolution)
	}

	fmt.Println()
	if isPreview {
		c.printSuccess(i18n.T("sync.preview_done"))
	} else {
		c.printSuccess(i18n.T("sync.import_done"))
	}
}

// formatFileSize returns a human-readable file size string.
func formatFileSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
