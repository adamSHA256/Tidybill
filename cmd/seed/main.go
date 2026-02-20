package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/adamSHA256/tidybill/internal/config"
)

func main() {
	lang := flag.String("lang", "cs", "Language preset: cs, sk, en")
	flag.Parse()

	data, ok := datasets[*lang]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown language: %s (available: cs, sk, en)\n", *lang)
		os.Exit(1)
	}

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dsn := cfg.DBPath + "?_pragma=foreign_keys(0)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check that migrations have been applied (schema_migrations table exists)
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableCount)
	if err != nil || tableCount == 0 {
		log.Fatal("Database not initialized. Run tidybill once first to apply migrations.")
	}

	fmt.Printf("Seeding database (%s) at: %s\n", *lang, cfg.DBPath)

	if err := clearData(db); err != nil {
		log.Fatalf("Failed to clear data: %v", err)
	}

	if err := seed(db, data); err != nil {
		log.Fatalf("Failed to seed: %v", err)
	}

	fmt.Println("Done! Seeded:")
	fmt.Printf("  - %d supplier(s) with bank accounts\n", len(data.Suppliers))
	fmt.Printf("  - %d customer(s)\n", len(data.Customers))
	fmt.Printf("  - %d catalog item(s)\n", len(data.Items))
	fmt.Printf("  - %d invoice(s) with line items\n", len(data.Invoices))
}

func clearData(db *sql.DB) error {
	tables := []string{
		"invoice_items", "invoices", "customer_items",
		"items", "bank_accounts", "customers", "suppliers",
	}
	for _, t := range tables {
		if _, err := db.Exec("DELETE FROM " + t); err != nil {
			return fmt.Errorf("clearing %s: %w", t, err)
		}
	}
	// Reset settings (language, units, payment_types) but keep dir.* overrides
	if _, err := db.Exec("DELETE FROM settings WHERE key NOT LIKE 'dir.%'"); err != nil {
		return fmt.Errorf("clearing settings: %w", err)
	}
	return nil
}

func seed(db *sql.DB, data dataset) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Settings
	for k, v := range data.Settings {
		if _, err := tx.Exec(
			"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
			k, v, v,
		); err != nil {
			return fmt.Errorf("setting %s: %w", k, err)
		}
	}

	// Suppliers + bank accounts
	for i := range data.Suppliers {
		s := &data.Suppliers[i]
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
		now := time.Now()
		if _, err := tx.Exec(`
			INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic,
				phone, email, website, logo_path, is_vat_payer, is_default, invoice_prefix, notes, language, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC,
			s.Phone, s.Email, s.Website, "", s.IsVATPayer, s.IsDefault, s.InvoicePrefix, s.Notes, s.Language, now, now,
		); err != nil {
			return fmt.Errorf("supplier %s: %w", s.Name, err)
		}

		for j := range s.BankAccounts {
			ba := &s.BankAccounts[j]
			if ba.ID == "" {
				ba.ID = uuid.New().String()
			}
			if _, err := tx.Exec(`
				INSERT INTO bank_accounts (id, supplier_id, name, account_number, iban, swift, currency, is_default, qr_type, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				ba.ID, s.ID, ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT, ba.Currency, ba.IsDefault, ba.QRType, now,
			); err != nil {
				return fmt.Errorf("bank account %s: %w", ba.Name, err)
			}
		}
	}

	// Customers
	for i := range data.Customers {
		c := &data.Customers[i]
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		now := time.Now()
		if _, err := tx.Exec(`
			INSERT INTO customers (id, name, street, city, zip, region, country, ico, dic,
				email, phone, default_vat_rate, default_due_days, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC,
			c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes, now, now,
		); err != nil {
			return fmt.Errorf("customer %s: %w", c.Name, err)
		}
	}

	// Items catalog — give items usage_count > 0 so they show in invoice creation
	for i := range data.Items {
		item := &data.Items[i]
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		now := time.Now()
		usageCount := i%5 + 1 // 1-5, so items appear in "most used"
		if _, err := tx.Exec(`
			INSERT INTO items (id, description, default_price, default_unit, default_vat_rate,
				category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
			item.Category, item.DefaultPrice, nil, usageCount, now, now,
		); err != nil {
			return fmt.Errorf("item %s: %w", item.Description, err)
		}
	}

	// Customer-item links — so items show as "customer items" during invoice creation
	for _, inv := range data.Invoices {
		customer := data.Customers[inv.customerIdx]
		for _, li := range inv.LineItems {
			// Find matching catalog item by description prefix
			for j := range data.Items {
				item := &data.Items[j]
				if len(li.Description) >= len(item.Description) && li.Description[:len(item.Description)] == item.Description {
					ciID := uuid.New().String()
					tx.Exec(`
						INSERT OR IGNORE INTO customer_items (id, customer_id, item_id, last_price, last_quantity, usage_count, last_used_at)
						VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)`,
						ciID, customer.ID, item.ID, li.UnitPrice, li.Quantity,
					)
					break
				}
			}
		}
	}

	// Invoices with line items
	for _, inv := range data.Invoices {
		if inv.ID == "" {
			inv.ID = uuid.New().String()
		}
		now := time.Now()

		// Resolve supplier & bank account by index
		supplier := data.Suppliers[inv.supplierIdx]
		customer := data.Customers[inv.customerIdx]
		bankAccountID := ""
		if inv.bankAccountIdx >= 0 && inv.supplierIdx < len(data.Suppliers) {
			accounts := data.Suppliers[inv.supplierIdx].BankAccounts
			if inv.bankAccountIdx < len(accounts) {
				bankAccountID = accounts[inv.bankAccountIdx].ID
			}
		}

		// Calculate totals from items
		var subtotal, vatTotal, total float64
		for _, li := range inv.LineItems {
			liSub := roundMoney(li.Quantity * li.UnitPrice)
			liVAT := roundMoney(liSub * li.VATRate / 100)
			subtotal += liSub
			vatTotal += liVAT
			total += roundMoney(liSub + liVAT)
		}

		var paidDate interface{}
		if inv.PaidDate != nil {
			paidDate = *inv.PaidDate
		}

		if _, err := tx.Exec(`
			INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id, status,
				issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol, currency,
				exchange_rate, subtotal, vat_total, total, notes, internal_notes, language, pdf_path, template_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			inv.ID, inv.InvoiceNumber, supplier.ID, customer.ID, bankAccountID, inv.Status,
			inv.IssueDate, inv.DueDate, paidDate, inv.IssueDate, inv.PaymentMethod, inv.VariableSymbol,
			inv.Currency, 1.0, roundMoney(subtotal), roundMoney(vatTotal), roundMoney(total),
			inv.Notes, "", inv.Language, "", "classic", now, now,
		); err != nil {
			return fmt.Errorf("invoice %s: %w", inv.InvoiceNumber, err)
		}

		// Line items
		for i, li := range inv.LineItems {
			liID := uuid.New().String()
			liSub := roundMoney(li.Quantity * li.UnitPrice)
			liVAT := roundMoney(liSub * li.VATRate / 100)
			liTotal := roundMoney(liSub + liVAT)
			if _, err := tx.Exec(`
				INSERT INTO invoice_items (id, invoice_id, description, quantity, unit,
					unit_price, vat_rate, subtotal, vat_amount, total, position)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				liID, inv.ID, li.Description, li.Quantity, li.Unit,
				li.UnitPrice, li.VATRate, liSub, liVAT, liTotal, i,
			); err != nil {
				return fmt.Errorf("invoice item %s/%d: %w", inv.InvoiceNumber, i, err)
			}
		}
	}

	return tx.Commit()
}

func roundMoney(amount float64) float64 {
	return float64(int(amount*100+0.5)) / 100
}

// ---------------------------------------------------------------------------
// Data structures
// ---------------------------------------------------------------------------

type seedSupplier struct {
	ID            string
	Name          string
	Street        string
	City          string
	ZIP           string
	Country       string
	ICO           string
	DIC           string
	Phone         string
	Email         string
	Website       string
	IsVATPayer    bool
	IsDefault     bool
	InvoicePrefix string
	Notes         string
	Language      string
	BankAccounts  []seedBankAccount
}

type seedBankAccount struct {
	ID            string
	Name          string
	AccountNumber string
	IBAN          string
	SWIFT         string
	Currency      string
	IsDefault     bool
	QRType        string
}

type seedCustomer struct {
	ID             string
	Name           string
	Street         string
	City           string
	ZIP            string
	Region         string
	Country        string
	ICO            string
	DIC            string
	Email          string
	Phone          string
	DefaultVATRate float64
	DefaultDueDays int
	Notes          string
}

type seedItem struct {
	ID             string
	Description    string
	DefaultPrice   float64
	DefaultUnit    string
	DefaultVATRate float64
	Category       string
}

type seedInvoice struct {
	ID             string
	InvoiceNumber  string
	Status         string
	IssueDate      time.Time
	DueDate        time.Time
	PaidDate       *time.Time
	PaymentMethod  string
	VariableSymbol string
	Currency       string
	Notes          string
	Language       string
	LineItems      []seedLineItem

	// Indexes into dataset slices
	supplierIdx    int
	customerIdx    int
	bankAccountIdx int // -1 = NULL (cash)
}

type seedLineItem struct {
	Description string
	Quantity    float64
	Unit        string
	UnitPrice   float64
	VATRate     float64
}

type dataset struct {
	Settings  map[string]string
	Suppliers []seedSupplier
	Customers []seedCustomer
	Items     []seedItem
	Invoices  []seedInvoice
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

func datePtr(year, month, day int) *time.Time {
	t := date(year, month, day)
	return &t
}

// ---------------------------------------------------------------------------
// Datasets
// ---------------------------------------------------------------------------

var datasets = map[string]dataset{
	"cs": csDataset(),
	"sk": skDataset(),
	"en": enDataset(),
}

func csDataset() dataset {
	return dataset{
		Settings: map[string]string{
			"language":      "cs",
			"units":         `["ks","hod","den","m²","m","kg","l"]`,
			"payment_types": `[{"code":"bank_transfer","is_default":true},{"code":"cash"},{"code":"card"}]`,
		},
		Suppliers: []seedSupplier{
			{
				Name: "Jan Novák - Webový design", Street: "Karlova 15", City: "Praha", ZIP: "11000",
				Country: "CZ", ICO: "12345678", DIC: "CZ12345678", Phone: "+420 601 123 456",
				Email: "jan@novak-web.cz", Website: "www.novak-web.cz", IsVATPayer: true,
				IsDefault: true, InvoicePrefix: "VF", Language: "cs",
				BankAccounts: []seedBankAccount{
					{Name: "Hlavní CZK", AccountNumber: "123456789/0100", IBAN: "CZ6508000000001234567890", SWIFT: "KOMBCZPP", Currency: "CZK", IsDefault: true, QRType: "spayd"},
					{Name: "EUR účet", AccountNumber: "987654321/0100", IBAN: "CZ6508000000009876543210", SWIFT: "KOMBCZPP", Currency: "EUR", IsDefault: true, QRType: "spayd"},
				},
			},
			{
				Name: "Petra Svobodová - Grafika", Street: "Masarykova 42", City: "Brno", ZIP: "60200",
				Country: "CZ", ICO: "87654321", Phone: "+420 602 987 654",
				Email: "petra@svobodova-grafika.cz", IsVATPayer: false,
				IsDefault: false, InvoicePrefix: "FA", Language: "cs",
				BankAccounts: []seedBankAccount{
					{Name: "Fio", AccountNumber: "2900123456/2010", IBAN: "CZ6520100000002900123456", SWIFT: "FIOBCZPP", Currency: "CZK", IsDefault: true, QRType: "spayd"},
				},
			},
		},
		Customers: []seedCustomer{
			{Name: "Acme s.r.o.", Street: "Průmyslová 10", City: "Praha", ZIP: "15000", Country: "CZ", ICO: "11111111", DIC: "CZ11111111", Email: "info@acme.cz", Phone: "+420 222 111 000", DefaultVATRate: 21, DefaultDueDays: 14},
			{Name: "Kořínek a syn", Street: "Náměstí Míru 3", City: "Olomouc", ZIP: "77900", Country: "CZ", ICO: "22222222", Email: "korinek@email.cz", DefaultDueDays: 30},
			{Name: "TechStart z.s.", Street: "Sokolská 88", City: "Ostrava", ZIP: "70200", Country: "CZ", ICO: "33333333", DIC: "CZ33333333", Email: "fakturace@techstart.cz", DefaultVATRate: 21, DefaultDueDays: 14},
			{Name: "Marie Dvořáková", Street: "Lipová 7", City: "Plzeň", ZIP: "30100", Country: "CZ", Email: "marie.dvorakova@email.cz", Phone: "+420 777 555 333", DefaultDueDays: 14},
			{Name: "Global Trade a.s.", Street: "Evropská 200", City: "Praha", ZIP: "16000", Country: "CZ", ICO: "44444444", DIC: "CZ44444444", Email: "invoices@globaltrade.cz", DefaultVATRate: 21, DefaultDueDays: 30},
			{Name: "Město Kutná Hora", Street: "Havlíčkovo nám. 552/1", City: "Kutná Hora", ZIP: "28401", Country: "CZ", ICO: "00236195", Email: "podatelna@mu.kutnahora.cz", DefaultDueDays: 21},
		},
		Items: []seedItem{
			{Description: "Tvorba webových stránek", DefaultPrice: 15000, DefaultUnit: "ks", DefaultVATRate: 21, Category: "web"},
			{Description: "Správa webových stránek (měsíc)", DefaultPrice: 2500, DefaultUnit: "ks", DefaultVATRate: 21, Category: "web"},
			{Description: "Grafický návrh loga", DefaultPrice: 8000, DefaultUnit: "ks", DefaultVATRate: 21, Category: "grafika"},
			{Description: "Programování na míru", DefaultPrice: 1200, DefaultUnit: "hod", DefaultVATRate: 21, Category: "vývoj"},
			{Description: "SEO optimalizace", DefaultPrice: 5000, DefaultUnit: "ks", DefaultVATRate: 21, Category: "marketing"},
			{Description: "Copywriting - článek", DefaultPrice: 1500, DefaultUnit: "ks", DefaultVATRate: 21, Category: "marketing"},
			{Description: "Konzultace", DefaultPrice: 800, DefaultUnit: "hod", DefaultVATRate: 21, Category: "služby"},
			{Description: "Hosting webu (rok)", DefaultPrice: 3000, DefaultUnit: "ks", DefaultVATRate: 21, Category: "web"},
			{Description: "Fotografování produktů", DefaultPrice: 500, DefaultUnit: "ks", DefaultVATRate: 21, Category: "grafika"},
			{Description: "Překlad CZ-EN", DefaultPrice: 350, DefaultUnit: "ks", DefaultVATRate: 21, Category: "služby"},
			{Description: "Tisk letáků A5", DefaultPrice: 5, DefaultUnit: "ks", DefaultVATRate: 21, Category: "tisk"},
			{Description: "IT podpora (den)", DefaultPrice: 6000, DefaultUnit: "den", DefaultVATRate: 21, Category: "služby"},
		},
		Invoices: []seedInvoice{
			{
				InvoiceNumber: "VF26-00001", Status: "paid", IssueDate: date(2026, 1, 5), DueDate: date(2026, 1, 19), PaidDate: datePtr(2026, 1, 15),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600001", Currency: "CZK", Language: "cs",
				Notes: "Děkujeme za spolupráci.", supplierIdx: 0, customerIdx: 0, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Tvorba webových stránek - eshop", Quantity: 1, Unit: "ks", UnitPrice: 45000, VATRate: 21},
					{Description: "SEO optimalizace", Quantity: 1, Unit: "ks", UnitPrice: 5000, VATRate: 21},
				},
			},
			{
				InvoiceNumber: "VF26-00002", Status: "paid", IssueDate: date(2026, 1, 15), DueDate: date(2026, 1, 29), PaidDate: datePtr(2026, 1, 28),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600002", Currency: "CZK", Language: "cs",
				supplierIdx: 0, customerIdx: 1, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Programování na míru - API modul", Quantity: 20, Unit: "hod", UnitPrice: 1200, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "VF26-00003", Status: "created", IssueDate: date(2026, 2, 1), DueDate: date(2026, 2, 15),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600003", Currency: "CZK", Language: "cs",
				supplierIdx: 0, customerIdx: 2, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Správa webových stránek (leden)", Quantity: 1, Unit: "ks", UnitPrice: 2500, VATRate: 21},
					{Description: "Hosting webu (rok 2026)", Quantity: 1, Unit: "ks", UnitPrice: 3000, VATRate: 21},
				},
			},
			{
				InvoiceNumber: "VF26-00004", Status: "overdue", IssueDate: date(2026, 1, 10), DueDate: date(2026, 1, 24),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600004", Currency: "CZK", Language: "cs",
				Notes: "Upomínka odeslána 2026-02-01.", supplierIdx: 0, customerIdx: 3, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Grafický návrh loga", Quantity: 1, Unit: "ks", UnitPrice: 8000, VATRate: 0},
					{Description: "Vizitky - návrh + tisk 200ks", Quantity: 1, Unit: "ks", UnitPrice: 3500, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "VF26-00005", Status: "draft", IssueDate: date(2026, 2, 18), DueDate: date(2026, 3, 4),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600005", Currency: "EUR", Language: "cs",
				supplierIdx: 0, customerIdx: 4, bankAccountIdx: 1,
				LineItems: []seedLineItem{
					{Description: "Webová aplikace - fáze 1", Quantity: 40, Unit: "hod", UnitPrice: 55, VATRate: 21},
					{Description: "UX konzultace", Quantity: 8, Unit: "hod", UnitPrice: 45, VATRate: 21},
				},
			},
			{
				InvoiceNumber: "FA26-00001", Status: "created", IssueDate: date(2026, 2, 10), DueDate: date(2026, 2, 24),
				PaymentMethod: "cash", VariableSymbol: "2600001", Currency: "CZK", Language: "cs",
				supplierIdx: 1, customerIdx: 5, bankAccountIdx: -1,
				LineItems: []seedLineItem{
					{Description: "Návrh propagačních materiálů", Quantity: 1, Unit: "ks", UnitPrice: 12000, VATRate: 0},
					{Description: "Fotografování akcí", Quantity: 3, Unit: "den", UnitPrice: 6000, VATRate: 0},
				},
			},
		},
	}
}

func skDataset() dataset {
	return dataset{
		Settings: map[string]string{
			"language":      "sk",
			"units":         `["ks","hod","deň","m²","m","kg","l"]`,
			"payment_types": `[{"code":"bank_transfer","is_default":true},{"code":"cash"},{"code":"card"}]`,
		},
		Suppliers: []seedSupplier{
			{
				Name: "Marek Horváth - IT služby", Street: "Hlavná 25", City: "Bratislava", ZIP: "81101",
				Country: "SK", ICO: "51234567", DIC: "SK2120123456", Phone: "+421 901 123 456",
				Email: "marek@horvath-it.sk", Website: "www.horvath-it.sk", IsVATPayer: true,
				IsDefault: true, InvoicePrefix: "FA", Language: "sk",
				BankAccounts: []seedBankAccount{
					{Name: "Hlavný EUR", AccountNumber: "SK3109000000005012345678", IBAN: "SK3109000000005012345678", SWIFT: "GIBASKBX", Currency: "EUR", IsDefault: true, QRType: "pay_by_square"},
				},
			},
			{
				Name: "Zuzana Kováčová - Preklady", Street: "Štúrova 18", City: "Košice", ZIP: "04001",
				Country: "SK", ICO: "41234567", Phone: "+421 902 987 654",
				Email: "zuzana@kovacova-preklad.sk", IsVATPayer: false,
				IsDefault: false, InvoicePrefix: "FA", Language: "sk",
				BankAccounts: []seedBankAccount{
					{Name: "Fio SK", AccountNumber: "SK8083300000002900123456", IBAN: "SK8083300000002900123456", SWIFT: "FIOZSKBA", Currency: "EUR", IsDefault: true, QRType: "pay_by_square"},
				},
			},
		},
		Customers: []seedCustomer{
			{Name: "Digitálne riešenia s.r.o.", Street: "Mlynské nivy 50", City: "Bratislava", ZIP: "82105", Country: "SK", ICO: "36123456", DIC: "SK2020123456", Email: "info@digires.sk", Phone: "+421 2 1234 5678", DefaultVATRate: 20, DefaultDueDays: 14},
			{Name: "Peter Baláž - SZČO", Street: "Námestie SNP 12", City: "Banská Bystrica", ZIP: "97401", Country: "SK", ICO: "43215678", Email: "peter.balaz@email.sk", DefaultDueDays: 14},
			{Name: "AutoServis Nitra s.r.o.", Street: "Cabajská 44", City: "Nitra", ZIP: "94901", Country: "SK", ICO: "36987654", DIC: "SK2020987654", Email: "fakturacia@autoservisnr.sk", DefaultVATRate: 20, DefaultDueDays: 30},
			{Name: "Slovenská Knižnica a.s.", Street: "Námestie slobody 1", City: "Martin", ZIP: "03601", Country: "SK", ICO: "00123456", Email: "objednavky@slk.sk", DefaultDueDays: 21},
			{Name: "Eva Tóthová", Street: "Komenského 8", City: "Prešov", ZIP: "08001", Country: "SK", Email: "eva.tothova@email.sk", Phone: "+421 911 222 333", DefaultDueDays: 14},
			{Name: "StartUp Hub Žilina z.z.p.o.", Street: "Mariánske námestie 2", City: "Žilina", ZIP: "01001", Country: "SK", ICO: "52345678", Email: "hub@startupza.sk", DefaultDueDays: 14},
		},
		Items: []seedItem{
			{Description: "Vývoj webovej aplikácie", DefaultPrice: 60, DefaultUnit: "hod", DefaultVATRate: 20, Category: "vývoj"},
			{Description: "Správa servera (mesiac)", DefaultPrice: 150, DefaultUnit: "ks", DefaultVATRate: 20, Category: "it"},
			{Description: "Návrh UI/UX", DefaultPrice: 50, DefaultUnit: "hod", DefaultVATRate: 20, Category: "dizajn"},
			{Description: "IT konzultácia", DefaultPrice: 40, DefaultUnit: "hod", DefaultVATRate: 20, Category: "it"},
			{Description: "Preklad SK-EN (normostrana)", DefaultPrice: 18, DefaultUnit: "ks", DefaultVATRate: 20, Category: "preklad"},
			{Description: "Preklad SK-DE (normostrana)", DefaultPrice: 20, DefaultUnit: "ks", DefaultVATRate: 20, Category: "preklad"},
			{Description: "Korektúra textu", DefaultPrice: 10, DefaultUnit: "ks", DefaultVATRate: 20, Category: "preklad"},
			{Description: "SEO audit webu", DefaultPrice: 300, DefaultUnit: "ks", DefaultVATRate: 20, Category: "marketing"},
			{Description: "Hosting webu (rok)", DefaultPrice: 120, DefaultUnit: "ks", DefaultVATRate: 20, Category: "it"},
			{Description: "Grafický návrh loga", DefaultPrice: 400, DefaultUnit: "ks", DefaultVATRate: 20, Category: "dizajn"},
			{Description: "Školenie IT bezpečnosť", DefaultPrice: 500, DefaultUnit: "deň", DefaultVATRate: 20, Category: "školenie"},
			{Description: "Technická podpora (deň)", DefaultPrice: 280, DefaultUnit: "deň", DefaultVATRate: 20, Category: "it"},
		},
		Invoices: []seedInvoice{
			{
				InvoiceNumber: "FA26-00001", Status: "paid", IssueDate: date(2026, 1, 8), DueDate: date(2026, 1, 22), PaidDate: datePtr(2026, 1, 20),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600001", Currency: "EUR", Language: "sk",
				Notes: "Ďakujeme za spoluprácu.", supplierIdx: 0, customerIdx: 0, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Vývoj webovej aplikácie - e-shop modul", Quantity: 80, Unit: "hod", UnitPrice: 60, VATRate: 20},
					{Description: "SEO audit webu", Quantity: 1, Unit: "ks", UnitPrice: 300, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "FA26-00002", Status: "paid", IssueDate: date(2026, 1, 20), DueDate: date(2026, 2, 3), PaidDate: datePtr(2026, 2, 1),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600002", Currency: "EUR", Language: "sk",
				supplierIdx: 0, customerIdx: 1, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "IT konzultácia - migrácia systému", Quantity: 16, Unit: "hod", UnitPrice: 40, VATRate: 0},
					{Description: "Správa servera (január)", Quantity: 1, Unit: "ks", UnitPrice: 150, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "FA26-00003", Status: "created", IssueDate: date(2026, 2, 1), DueDate: date(2026, 3, 3),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600003", Currency: "EUR", Language: "sk",
				supplierIdx: 0, customerIdx: 2, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Vývoj interného systému - fáza 2", Quantity: 60, Unit: "hod", UnitPrice: 60, VATRate: 20},
					{Description: "Hosting webu (rok 2026)", Quantity: 1, Unit: "ks", UnitPrice: 120, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "FA26-00004", Status: "overdue", IssueDate: date(2026, 1, 12), DueDate: date(2026, 1, 26),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600004", Currency: "EUR", Language: "sk",
				Notes: "Upomienka odoslaná 2026-02-03.", supplierIdx: 0, customerIdx: 4, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Návrh UI/UX - mobilná aplikácia", Quantity: 24, Unit: "hod", UnitPrice: 50, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "FA26-00005", Status: "draft", IssueDate: date(2026, 2, 15), DueDate: date(2026, 3, 1),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600005", Currency: "EUR", Language: "sk",
				supplierIdx: 0, customerIdx: 5, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Školenie IT bezpečnosť", Quantity: 2, Unit: "deň", UnitPrice: 500, VATRate: 20},
					{Description: "Príprava materiálov", Quantity: 8, Unit: "hod", UnitPrice: 40, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "FA26-00001", Status: "created", IssueDate: date(2026, 2, 5), DueDate: date(2026, 2, 19),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600001", Currency: "EUR", Language: "sk",
				supplierIdx: 1, customerIdx: 3, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Preklad SK-EN (výročná správa)", Quantity: 45, Unit: "ks", UnitPrice: 18, VATRate: 0},
					{Description: "Korektúra textu", Quantity: 45, Unit: "ks", UnitPrice: 10, VATRate: 0},
				},
			},
		},
	}
}

func enDataset() dataset {
	return dataset{
		Settings: map[string]string{
			"language":      "en",
			"units":         `["pcs","hr","day","m²","m","kg","l"]`,
			"payment_types": `[{"code":"bank_transfer","is_default":true},{"code":"cash"},{"code":"card"},{"code":"paypal"}]`,
		},
		Suppliers: []seedSupplier{
			{
				Name: "Smith & Co. Digital", Street: "45 Baker Street", City: "London", ZIP: "W1U 8EW",
				Country: "GB", ICO: "12345678", DIC: "GB123456789", Phone: "+44 20 7946 0958",
				Email: "hello@smithdigital.co.uk", Website: "www.smithdigital.co.uk", IsVATPayer: true,
				IsDefault: true, InvoicePrefix: "INV", Language: "en",
				BankAccounts: []seedBankAccount{
					{Name: "Main GBP", AccountNumber: "12345678", IBAN: "GB29NWBK60161331926819", SWIFT: "NWBKGB2L", Currency: "GBP", IsDefault: true, QRType: "none"},
					{Name: "EUR Account", IBAN: "GB82WEST12345698765432", SWIFT: "NWBKGB2L", Currency: "EUR", IsDefault: true, QRType: "epc"},
				},
			},
			{
				Name: "Freelance Jane Doe", Street: "12 Elm Avenue", City: "Manchester", ZIP: "M1 1AA",
				Country: "GB", ICO: "87654321", Phone: "+44 161 123 4567",
				Email: "jane@janedoe-design.com", IsVATPayer: false,
				IsDefault: false, InvoicePrefix: "JD", Language: "en",
				BankAccounts: []seedBankAccount{
					{Name: "Monzo", AccountNumber: "98765432", IBAN: "GB33BUKB20201555555555", SWIFT: "BUKBGB22", Currency: "GBP", IsDefault: true, QRType: "none"},
				},
			},
		},
		Customers: []seedCustomer{
			{Name: "TechVentures Ltd", Street: "100 Silicon Way", City: "Cambridge", ZIP: "CB1 2AB", Country: "GB", ICO: "09876543", DIC: "GB987654321", Email: "accounts@techventures.co.uk", Phone: "+44 1223 456 789", DefaultVATRate: 20, DefaultDueDays: 30},
			{Name: "Green Garden Services", Street: "8 Park Lane", City: "Oxford", ZIP: "OX1 3QD", Country: "GB", ICO: "11223344", Email: "billing@greengarden.co.uk", DefaultDueDays: 14},
			{Name: "Northern Brewery Co.", Street: "55 Ale Road", City: "Edinburgh", ZIP: "EH1 1RE", Country: "GB", ICO: "55667788", DIC: "GB556677889", Email: "invoices@northernbrewery.co.uk", DefaultVATRate: 20, DefaultDueDays: 30},
			{Name: "Sarah Williams", Street: "3 Rose Cottage", City: "Bristol", ZIP: "BS1 5TJ", Country: "GB", Email: "sarah.williams@email.com", Phone: "+44 7700 900123", DefaultDueDays: 14},
			{Name: "EuroTrade GmbH", Street: "Friedrichstraße 100", City: "Berlin", ZIP: "10117", Country: "DE", ICO: "HRB12345", DIC: "DE123456789", Email: "invoices@eurotrade.de", DefaultVATRate: 0, DefaultDueDays: 30},
			{Name: "City Council of Bath", Street: "Guildhall, High Street", City: "Bath", ZIP: "BA1 5AW", Country: "GB", ICO: "00112233", Email: "procurement@bathnes.gov.uk", DefaultDueDays: 30},
		},
		Items: []seedItem{
			{Description: "Website development", DefaultPrice: 750, DefaultUnit: "day", DefaultVATRate: 20, Category: "web"},
			{Description: "Website maintenance (monthly)", DefaultPrice: 200, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "web"},
			{Description: "Logo design", DefaultPrice: 500, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "design"},
			{Description: "Custom software development", DefaultPrice: 95, DefaultUnit: "hr", DefaultVATRate: 20, Category: "development"},
			{Description: "SEO optimization package", DefaultPrice: 400, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "marketing"},
			{Description: "Copywriting - blog article", DefaultPrice: 120, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "marketing"},
			{Description: "Consultation", DefaultPrice: 75, DefaultUnit: "hr", DefaultVATRate: 20, Category: "services"},
			{Description: "Web hosting (annual)", DefaultPrice: 240, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "web"},
			{Description: "Product photography", DefaultPrice: 45, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "design"},
			{Description: "Translation EN-DE", DefaultPrice: 30, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "services"},
			{Description: "Leaflet printing A5 (100pcs)", DefaultPrice: 35, DefaultUnit: "pcs", DefaultVATRate: 20, Category: "print"},
			{Description: "IT support (daily rate)", DefaultPrice: 450, DefaultUnit: "day", DefaultVATRate: 20, Category: "services"},
		},
		Invoices: []seedInvoice{
			{
				InvoiceNumber: "INV26-00001", Status: "paid", IssueDate: date(2026, 1, 3), DueDate: date(2026, 2, 2), PaidDate: datePtr(2026, 1, 25),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600001", Currency: "GBP", Language: "en",
				Notes: "Thank you for your business.", supplierIdx: 0, customerIdx: 0, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Website development - e-commerce platform", Quantity: 12, Unit: "day", UnitPrice: 750, VATRate: 20},
					{Description: "SEO optimization package", Quantity: 1, Unit: "pcs", UnitPrice: 400, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "INV26-00002", Status: "paid", IssueDate: date(2026, 1, 18), DueDate: date(2026, 2, 1), PaidDate: datePtr(2026, 1, 30),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600002", Currency: "GBP", Language: "en",
				supplierIdx: 0, customerIdx: 1, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Custom software - inventory module", Quantity: 40, Unit: "hr", UnitPrice: 95, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "INV26-00003", Status: "created", IssueDate: date(2026, 2, 1), DueDate: date(2026, 3, 3),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600003", Currency: "GBP", Language: "en",
				supplierIdx: 0, customerIdx: 2, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Website maintenance (January)", Quantity: 1, Unit: "pcs", UnitPrice: 200, VATRate: 20},
					{Description: "Web hosting (annual 2026)", Quantity: 1, Unit: "pcs", UnitPrice: 240, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "INV26-00004", Status: "overdue", IssueDate: date(2026, 1, 8), DueDate: date(2026, 1, 22),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600004", Currency: "GBP", Language: "en",
				Notes: "Reminder sent on 2026-02-01.", supplierIdx: 0, customerIdx: 3, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Logo design - full branding package", Quantity: 1, Unit: "pcs", UnitPrice: 1200, VATRate: 0},
					{Description: "Business card design + print 250pcs", Quantity: 1, Unit: "pcs", UnitPrice: 180, VATRate: 0},
				},
			},
			{
				InvoiceNumber: "INV26-00005", Status: "draft", IssueDate: date(2026, 2, 18), DueDate: date(2026, 3, 20),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600005", Currency: "EUR", Language: "en",
				supplierIdx: 0, customerIdx: 4, bankAccountIdx: 1,
				LineItems: []seedLineItem{
					{Description: "Web application - phase 1", Quantity: 8, Unit: "day", UnitPrice: 750, VATRate: 20},
					{Description: "UX consultation", Quantity: 16, Unit: "hr", UnitPrice: 75, VATRate: 20},
				},
			},
			{
				InvoiceNumber: "JD26-00001", Status: "created", IssueDate: date(2026, 2, 8), DueDate: date(2026, 3, 10),
				PaymentMethod: "bank_transfer", VariableSymbol: "2600001", Currency: "GBP", Language: "en",
				supplierIdx: 1, customerIdx: 5, bankAccountIdx: 0,
				LineItems: []seedLineItem{
					{Description: "Event branding design package", Quantity: 1, Unit: "pcs", UnitPrice: 800, VATRate: 0},
					{Description: "Product photography", Quantity: 25, Unit: "pcs", UnitPrice: 45, VATRate: 0},
				},
			},
		},
	}
}
