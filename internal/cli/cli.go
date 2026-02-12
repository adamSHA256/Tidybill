package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/service"
)

type CLI struct {
	db          *database.DB
	cfg         *config.Config
	suppliers   *repository.SupplierRepository
	customers   *repository.CustomerRepository
	bankAccs    *repository.BankAccountRepository
	invoices    *repository.InvoiceRepository
	invItems    *repository.InvoiceItemRepository
	pdfService  *service.PDFService
	settings    *repository.SettingsRepository
	scanner     *bufio.Scanner
	currentSupp string // Current supplier ID
}

func New(db *database.DB, cfg *config.Config) *CLI {
	return &CLI{
		db:         db,
		cfg:        cfg,
		suppliers:  repository.NewSupplierRepository(db.DB),
		customers:  repository.NewCustomerRepository(db.DB),
		bankAccs:   repository.NewBankAccountRepository(db.DB),
		invoices:   repository.NewInvoiceRepository(db.DB),
		invItems:   repository.NewInvoiceItemRepository(db.DB),
		pdfService: service.NewPDFService(cfg.PDFDir),
		settings:   repository.NewSettingsRepository(db.DB),
		scanner:    bufio.NewScanner(os.Stdin),
	}
}

func (c *CLI) Run() error {
	// Load saved language
	if lang, err := c.settings.Get("language"); err == nil && lang != "" {
		i18n.SetLang(i18n.Lang(lang))
	}

	// Check if first run
	empty, err := c.db.IsEmpty()
	if err != nil {
		return err
	}

	if empty {
		if err := c.firstRunWizard(); err != nil {
			return err
		}
	}

	// Load default supplier
	supplier, err := c.suppliers.GetDefault()
	if err != nil {
		return err
	}
	if supplier != nil {
		c.currentSupp = supplier.ID
	}

	return c.mainMenu()
}

func (c *CLI) mainMenu() error {
	for {
		c.clearScreen()
		c.printHeader()

		unpaid, _ := c.invoices.CountUnpaid()
		overdue, _ := c.invoices.CountOverdue()

		fmt.Printf("  1) %s\n", i18n.T("menu.create_invoice"))
		fmt.Printf("  2) %s\n", i18n.T("menu.create_from_existing"))
		fmt.Printf("  3) %s\n", i18n.T("menu.list_invoices"))
		if unpaid > 0 {
			fmt.Printf("  4) %s [%s", i18n.T("menu.unpaid_invoices"), i18n.Tf("menu.unpaid_count", unpaid))
			if overdue > 0 {
				fmt.Printf(", %s", i18n.Tf("menu.overdue_count", overdue))
			}
			fmt.Println("]")
		} else {
			fmt.Printf("  4) %s\n", i18n.T("menu.unpaid_invoices"))
		}
		fmt.Printf("  5) %s\n", i18n.T("menu.customers"))
		fmt.Printf("  6) %s\n", i18n.T("menu.items_catalog"))
		fmt.Printf("  7) %s\n", i18n.T("menu.suppliers"))
		fmt.Printf("  8) %s\n", i18n.T("menu.sync"))
		fmt.Printf("  9) %s\n", i18n.T("menu.pdf_templates"))
		fmt.Printf("  S) %s\n", i18n.T("menu.settings"))
		fmt.Printf("  W) %s\n", i18n.T("menu.overview"))
		fmt.Printf("  0) %s\n", i18n.T("menu.quit"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "1":
			c.createInvoice()
		case "2":
			c.createFromExisting()
		case "3":
			c.listInvoices()
		case "4":
			c.listUnpaidInvoices()
		case "5":
			c.customersMenu()
		case "6":
			c.itemsMenu()
		case "7":
			c.suppliersMenu()
		case "8":
			c.syncMenu()
		case "9":
			c.templatesMenu()
		case "s":
			c.settingsMenu()
		case "w":
			c.showStats()
		case "0", "q":
			fmt.Println(i18n.T("app.goodbye"))
			return nil
		}
	}
}

func (c *CLI) printHeader() {
	supplier, _ := c.suppliers.GetDefault()
	name := i18n.T("header.no_company")
	if supplier != nil {
		name = supplier.Name
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      TIDYBILL v0.1                         ║")
	fmt.Printf("║  %s %-50s ║\n", i18n.T("header.company"), name)
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                            ║")
}

func (c *CLI) printFooter() {
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

func (c *CLI) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// ErrGoBack is returned when user wants to go back
var ErrGoBack = fmt.Errorf("go back")

func (c *CLI) prompt(label string) string {
	fmt.Printf("%s: ", label)
	c.scanner.Scan()
	return strings.TrimSpace(c.scanner.Text())
}

// promptWithBack returns the input and true if user wants to go back (entered 0 or q)
func (c *CLI) promptWithBack(label string) (string, bool) {
	fmt.Printf("%s %s: ", label, i18n.T("prompt.back_hint"))
	c.scanner.Scan()
	val := strings.TrimSpace(c.scanner.Text())
	if val == "0" || strings.ToLower(val) == "q" {
		return "", true
	}
	return val, false
}

func (c *CLI) promptDefault(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	c.scanner.Scan()
	val := strings.TrimSpace(c.scanner.Text())
	if val == "" {
		return defaultVal
	}
	return val
}

func (c *CLI) promptFloat(label string, defaultVal float64) float64 {
	var defStr string
	if defaultVal != 0 {
		defStr = fmt.Sprintf("%.2f", defaultVal)
	}
	str := c.promptDefault(label, defStr)
	if str == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(strings.ReplaceAll(str, ",", "."), 64)
	if err != nil {
		return defaultVal
	}
	return val
}

func (c *CLI) promptInt(label string, defaultVal int) int {
	var defStr string
	if defaultVal != 0 {
		defStr = strconv.Itoa(defaultVal)
	}
	str := c.promptDefault(label, defStr)
	if str == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal
	}
	return val
}

func (c *CLI) confirm(message string) bool {
	response := c.promptDefault(message+" "+i18n.T("confirm.yes_no"), i18n.T("confirm.yes_default"))
	lower := strings.ToLower(response)
	return lower == "a" || lower == "y" || response == ""
}

func (c *CLI) waitEnter() {
	fmt.Println()
	fmt.Print(i18n.T("prompt.press_enter"))
	c.scanner.Scan()
}

func (c *CLI) printError(msg string) {
	fmt.Printf("\n❌ %s %s\n", i18n.T("error.prefix"), msg)
}

func (c *CLI) printSuccess(msg string) {
	fmt.Printf("\n✓ %s\n", msg)
}

func (c *CLI) showStats() {
	count, err := c.suppliers.Count()
	if err != nil {
		fmt.Println(i18n.T("error.stats_suppliers"), err)
	} else {
		fmt.Println(i18n.Tf("stats.suppliers_count", count))
	}
	invoices, err := c.invoices.CountUnpaid()
	if err != nil {
		fmt.Println(i18n.T("error.stats_invoices"), err)
	} else {
		fmt.Println(i18n.Tf("stats.unpaid_invoices_count", invoices))
	}
	c.waitEnter()
}
