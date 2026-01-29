package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/user/invoice-app/internal/config"
	"github.com/user/invoice-app/internal/database"
	"github.com/user/invoice-app/internal/database/repository"
	"github.com/user/invoice-app/internal/service"
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
		scanner:    bufio.NewScanner(os.Stdin),
	}
}

func (c *CLI) Run() error {
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

		fmt.Println("  1) Vytvořit novou fakturu")
		fmt.Println("  2) Vytvořit fakturu z existující")
		fmt.Println("  3) Seznam faktur")
		if unpaid > 0 {
			fmt.Printf("  4) Nezaplacené faktury              [%d nezaplacených", unpaid)
			if overdue > 0 {
				fmt.Printf(", %d po splatnosti", overdue)
			}
			fmt.Println("]")
		} else {
			fmt.Println("  4) Nezaplacené faktury")
		}
		fmt.Println("  5) Odběratelé")
		fmt.Println("  6) Katalog položek")
		fmt.Println("  7) Dodavatelé (vaše firmy)")
		fmt.Println("  8) Sync / Import / Export")
		fmt.Println("  9) Šablony PDF")
		fmt.Println("  S) Nastavení")
		fmt.Println("  0) Ukončit")
		fmt.Println()

		choice := c.prompt("Vyberte možnost")

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
		case "0", "q":
			fmt.Println("Na shledanou!")
			return nil
		}
	}
}

func (c *CLI) printHeader() {
	supplier, _ := c.suppliers.GetDefault()
	name := "Není nastaveno"
	if supplier != nil {
		name = supplier.Name
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    INVOICE MANAGER v0.1                    ║")
	fmt.Printf("║  Firma: %-50s ║\n", name)
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
	fmt.Printf("%s (0=zpět): ", label)
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
	response := c.promptDefault(message+" [A/n]", "a")
	return strings.ToLower(response) == "a" || strings.ToLower(response) == "y" || response == ""
}

func (c *CLI) waitEnter() {
	fmt.Println()
	fmt.Print("Stiskněte Enter pro pokračování...")
	c.scanner.Scan()
}

func (c *CLI) printError(msg string) {
	fmt.Printf("\n❌ Chyba: %s\n", msg)
}

func (c *CLI) printSuccess(msg string) {
	fmt.Printf("\n✓ %s\n", msg)
}
