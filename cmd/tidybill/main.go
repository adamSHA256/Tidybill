package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adamSHA256/tidybill/internal/api"
	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cli"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

// initDB loads config, opens the database, applies settings, and loads language.
// It returns everything needed by export/import/gui/cli modes.
func initDB() (*config.Config, *database.DB, *repository.SettingsRepository) {
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("app.error_config", err))
		os.Exit(1)
	}

	db, err := database.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("app.error_database", err))
		os.Exit(1)
	}

	settings := repository.NewSettingsRepository(db.DB)
	if err := cfg.ApplySettings(settings.Get); err != nil {
		log.Printf("Warning: failed to apply settings: %v", err)
	}

	if lang, err := settings.Get("language"); err == nil && lang != "" {
		i18n.SetLang(i18n.Lang(lang))
	}

	// Persist email template defaults if not already set
	settings.SetDefault("email.default_subject", "Faktura ((number))")
	settings.SetDefault("email.default_body", "Dobrý den,\n\nv příloze zasílám fakturu č. ((number)) na částku ((total)).\nSplatnost: ((due_date)).\n\nS pozdravem\n((supplier))")
	settings.SetDefault("email.copy_subject", "TidyBill - ((subject))")

	return cfg, db, settings
}

func handleExport(exportPath string, passphrase string) {
	_, db, settings := initDB()
	defer db.Close()

	fmt.Printf("  Exporting to %s...\n", exportPath)

	svc := backup.NewExportService(
		db.DB,
		repository.NewSupplierRepository(db.DB),
		repository.NewBankAccountRepository(db.DB),
		repository.NewCustomerRepository(db.DB),
		repository.NewInvoiceRepository(db.DB),
		repository.NewInvoiceItemRepository(db.DB),
		repository.NewItemRepository(db.DB),
		repository.NewCustomerItemRepository(db.DB),
		repository.NewPDFTemplateRepository(db.DB),
		repository.NewSmtpConfigRepository(db.DB),
		settings,
	)

	var data []byte
	var err error
	if passphrase != "" {
		fmt.Println("  Encrypting export...")
		data, err = svc.ExportEncryptedJSON(nil, passphrase)
	} else {
		data, err = svc.ExportJSON(nil)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Decode to get counts for the summary line.
	file, _ := svc.Export(nil)
	if file != nil {
		fmt.Printf("  \u2713 %d suppliers, %d bank accounts, %d customers, %d invoices, %d items\n",
			len(file.Suppliers), len(file.BankAccounts), len(file.Customers),
			len(file.Invoices), len(file.InvoiceItems))
	}

	info, err := os.Stat(exportPath)
	if err == nil {
		sizeKB := info.Size() / 1024
		if sizeKB == 0 {
			sizeKB = 1
		}
		fmt.Printf("  \u2713 Export complete (%d KB)\n", sizeKB)
	} else {
		fmt.Println("  \u2713 Export complete")
	}
}

func handleImport(importPath string, mode string, preview bool, passphrase string) {
	_, db, _ := initDB()
	defer db.Close()

	fmt.Printf("  Importing from %s...\n", importPath)

	f, err := os.Open(importPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	svc := backup.NewImportService(db.DB)

	opts := backup.ImportOptions{
		Passphrase:            passphrase,
		InvoiceNumberConflict: "skip",
	}

	if preview {
		opts.Mode = backup.ImportModePreview
	} else {
		switch strings.ToLower(mode) {
		case "merge":
			opts.Mode = backup.ImportModeSmartMerge
		case "replace":
			opts.Mode = backup.ImportModeFullReplace
		case "force":
			opts.Mode = backup.ImportModeForce
		default:
			fmt.Fprintf(os.Stderr, "  Unknown import mode: %s (use merge, replace, or force)\n", mode)
			os.Exit(1)
		}
	}

	modeName := map[string]string{
		backup.ImportModeSmartMerge:  "smart merge",
		backup.ImportModeFullReplace: "full replace",
		backup.ImportModeForce:       "force",
		backup.ImportModePreview:     "preview (dry run)",
	}
	fmt.Printf("  Mode: %s\n", modeName[opts.Mode])

	report, err := svc.Import(f, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		os.Exit(1)
	}

	// Print table
	tables := []struct {
		label string
		key   string
	}{
		{"Suppliers", "suppliers"},
		{"Bank Accounts", "bank_accounts"},
		{"Customers", "customers"},
		{"Items", "items"},
		{"Invoices", "invoices"},
		{"Invoice Items", "invoice_items"},
		{"Settings", "settings"},
		{"VAT Rates", "vat_rates"},
		{"PDF Templates", "pdf_templates"},
		{"SMTP Configs", "smtp_configs"},
		{"Customer Items", "customer_items"},
	}

	// Filter to only tables with data.
	type row struct {
		label                      string
		insert, update, skip int
	}
	var rows []row
	maxLabel := 5 // minimum column width
	for _, t := range tables {
		s, ok := report.Details[t.key]
		if !ok || (s.Insert == 0 && s.Update == 0 && s.Skip == 0) {
			continue
		}
		if len(t.label) > maxLabel {
			maxLabel = len(t.label)
		}
		rows = append(rows, row{t.label, s.Insert, s.Update, s.Skip})
	}

	if len(rows) > 0 {
		hdrFmt := fmt.Sprintf("  \u250c\u2500%%-%ds\u2500\u252c\u2500%%8s\u2500\u252c\u2500%%7s\u2500\u252c\u2500%%7s\u2500\u2510\n", maxLabel)
		rowFmt := fmt.Sprintf("  \u2502 %%-%ds \u2502 %%8d \u2502 %%7d \u2502 %%7d \u2502\n", maxLabel)
		sepFmt := fmt.Sprintf("  \u251c\u2500%%-%ds\u2500\u253c\u2500%%8s\u2500\u253c\u2500%%7s\u2500\u253c\u2500%%7s\u2500\u2524\n", maxLabel)
		botFmt := fmt.Sprintf("  \u2514\u2500%%-%ds\u2500\u2534\u2500%%8s\u2500\u2534\u2500%%7s\u2500\u2534\u2500%%7s\u2500\u2518\n", maxLabel)

		pad := func(n int, ch string) string {
			s := ""
			for i := 0; i < n; i++ {
				s += ch
			}
			return s
		}

		fmt.Printf(hdrFmt, pad(maxLabel, "\u2500"), pad(8, "\u2500"), pad(7, "\u2500"), pad(7, "\u2500"))
		fmt.Printf(fmt.Sprintf("  \u2502 %%-%ds \u2502 %%8s \u2502 %%7s \u2502 %%7s \u2502\n", maxLabel),
			"Table", "Inserted", "Updated", "Skipped")
		fmt.Printf(sepFmt, pad(maxLabel, "\u2500"), pad(8, "\u2500"), pad(7, "\u2500"), pad(7, "\u2500"))
		for _, r := range rows {
			fmt.Printf(rowFmt, r.label, r.insert, r.update, r.skip)
		}
		fmt.Printf(botFmt, pad(maxLabel, "\u2500"), pad(8, "\u2500"), pad(7, "\u2500"), pad(7, "\u2500"))
	}

	for _, c := range report.Conflicts {
		fmt.Printf("  \u26a0 Conflict: %s %s (%s)\n", c.Table, c.Description, c.Resolution)
	}
	for _, w := range report.Warnings {
		fmt.Printf("  \u26a0 Warning: %s %s (%s)\n", w.Table, w.Description, w.Resolution)
	}

	if preview {
		fmt.Println("  \u2713 Preview complete (no changes made)")
	} else {
		fmt.Println("  \u2713 Import complete")
	}
}

func main() {
	gui := flag.Bool("gui", false, "Start web UI mode (HTTP server)")
	port := flag.String("port", "8080", "HTTP server port (used with --gui)")
	parentPID := flag.Int("parent-pid", 0, "Parent process PID (exit when parent dies)")
	exportFile := flag.String("export", "", "Export all data to a .tidybill file")
	importFile := flag.String("import", "", "Import data from a .tidybill file")
	importMode := flag.String("mode", "merge", "Import mode: merge, replace, force")
	importPreview := flag.Bool("preview", false, "Preview import without making changes")
	passphrase := flag.String("passphrase", "", "Passphrase for encrypted export/import")
	flag.Parse()

	// Handle export
	if *exportFile != "" {
		handleExport(*exportFile, *passphrase)
		os.Exit(0)
	}

	// Handle import
	if *importFile != "" {
		handleImport(*importFile, *importMode, *importPreview, *passphrase)
		os.Exit(0)
	}

	// Load configuration
	cfg, db, _ := initDB()
	defer db.Close()

	if *gui {
		// Web UI mode
		srv := api.NewServer(db.DB, cfg)

		listener, err := net.Listen("tcp", "127.0.0.1:"+*port)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
		actualPort := listener.Addr().(*net.TCPAddr).Port
		fmt.Printf("TIDYBILL_PORT=%d\n", actualPort)

		httpServer := &http.Server{Handler: srv.Router()}

		shutdown := make(chan struct{})

		// Handle SIGINT/SIGTERM — immediate shutdown for sidecar
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			log.Println("[tidybill] shutting down...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			httpServer.Shutdown(shutdownCtx)
			close(shutdown)
		}()

		// Watch parent process — exit if it dies (covers crashes)
		if *parentPID > 0 {
			go func() {
				for {
					time.Sleep(2 * time.Second)
					if !isProcessAlive(*parentPID) {
						log.Println("[tidybill] parent process gone, shutting down")
						httpServer.Shutdown(context.Background())
						close(shutdown)
						return
					}
				}
			}()
		}

		// Start server in background
		go func() {
			log.Printf("TidyBill API server running on http://localhost:%d", actualPort)
			if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed: %v", err)
			}
		}()

		<-shutdown
	} else {
		// CLI mode — double Ctrl+C within 3s to exit
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT)
		go func() {
			for {
				<-sigCh
				fmt.Fprintln(os.Stderr, "\nPress Ctrl+C again within 3s to exit")
				select {
				case <-sigCh:
					os.Exit(0)
				case <-time.After(3 * time.Second):
					// Reset — next Ctrl+C will warn again
				}
			}
		}()

		app := cli.New(db, cfg)
		if err := app.Run(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.Tf("app.error_general", err))
			os.Exit(1)
		}
	}
}

