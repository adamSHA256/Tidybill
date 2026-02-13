package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	items       *repository.ItemRepository
	custItems   *repository.CustomerItemRepository
	templates   *repository.PDFTemplateRepository
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
		pdfService: service.NewPDFService(cfg.PDFDir, cfg.PreviewDir),
		settings:   repository.NewSettingsRepository(db.DB),
		items:      repository.NewItemRepository(db.DB),
		custItems:  repository.NewCustomerItemRepository(db.DB),
		templates:  repository.NewPDFTemplateRepository(db.DB),
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

	c.invoices.MarkOverdue()

	return c.mainMenu()
}

func (c *CLI) mainMenu() error {
	for {
		c.invoices.MarkOverdue()
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

func (c *CLI) promptMaxLen(label string, maxLen int) string {
	for {
		val := c.prompt(label)
		if len([]rune(val)) <= maxLen {
			return val
		}
		fmt.Printf("  %s %s\n", i18n.T("error.prefix"),
			i18n.Tf("error.input_too_long", maxLen))
	}
}

func (c *CLI) promptMaxLenWithBack(label string, maxLen int) (string, bool) {
	for {
		val, goBack := c.promptWithBack(label)
		if goBack {
			return "", true
		}
		if len([]rune(val)) <= maxLen {
			return val, false
		}
		fmt.Printf("  %s %s\n", i18n.T("error.prefix"),
			i18n.Tf("error.input_too_long", maxLen))
	}
}

func (c *CLI) promptDefaultMaxLen(label, defaultVal string, maxLen int) string {
	for {
		val := c.promptDefault(label, defaultVal)
		if len([]rune(val)) <= maxLen {
			return val
		}
		fmt.Printf("  %s %s\n", i18n.T("error.prefix"),
			i18n.Tf("error.input_too_long", maxLen))
	}
}

// printMultiline prints a label+value where continuation lines align under the first.
// prefix is the indent before the label (e.g. "  "), label is the i18n format like "Poznámky:  %s".
func (c *CLI) printMultiline(prefix, label, value string) {
	// Find where %s sits in the label to compute padding width
	idx := strings.Index(label, "%s")
	if idx < 0 {
		fmt.Printf(prefix+label+"\n", value)
		return
	}
	pad := strings.Repeat(" ", len(prefix)+idx)
	aligned := strings.ReplaceAll(value, "\n", "\n"+pad)
	fmt.Printf(prefix+label+"\n", aligned)
}

// editNotes opens the current notes in $EDITOR so the user can freely modify them.
// Returns the updated notes string.
func (c *CLI) editNotes(current string) string {
	// Create temp file with current content
	tmpFile, err := os.CreateTemp("", "tidybill-notes-*.txt")
	if err != nil {
		c.printError(err.Error())
		return current
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(current); err != nil {
		tmpFile.Close()
		c.printError(err.Error())
		return current
	}
	tmpFile.Close()

	// Find editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"nano", "vim", "vi"} {
			if p, err := exec.LookPath(e); err == nil {
				editor = p
				break
			}
		}
	}
	if editor == "" {
		c.printError("no editor found, set $EDITOR")
		return current
	}

	// Resolve editor path for exec
	editorPath, err := exec.LookPath(filepath.Base(editor))
	if err != nil {
		editorPath = editor
	}

	// Open editor
	cmd := exec.Command(editorPath, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		c.printError(err.Error())
		return current
	}

	// Read back
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		c.printError(err.Error())
		return current
	}

	return strings.TrimRight(string(content), "\n\r ")
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
