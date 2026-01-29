package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/user/invoice-app/internal/database/repository"
	"github.com/user/invoice-app/internal/model"
	"github.com/user/invoice-app/internal/service"
)

func (c *CLI) createInvoice() {
	c.clearScreen()
	fmt.Println("=== NOVÁ FAKTURA ===")
	fmt.Println("(Kdykoliv zadejte 0 pro návrat zpět)")
	fmt.Println()

	// Select supplier (if more than one)
	supplier, goBack := c.selectSupplierForInvoice()
	if goBack || supplier == nil {
		return
	}

	// Select customer
	customer, goBack := c.selectCustomerWithBack()
	if goBack || customer == nil {
		return
	}

	// Select bank account (if more than one)
	bankAcc, goBack := c.selectBankAccountForInvoice(supplier.ID)
	if goBack || bankAcc == nil {
		return
	}

	// Generate invoice number
	invNumber, err := c.invoices.GetNextNumber(supplier.ID, supplier.InvoicePrefix)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Create invoice
	invoice := model.NewInvoice(supplier.ID, customer.ID, bankAcc.ID)
	invoice.InvoiceNumber = invNumber
	invoice.VariableSymbol = repository.GenerateVariableSymbol(invNumber)
	invoice.Currency = bankAcc.Currency
	invoice.DueDate = time.Now().AddDate(0, 0, customer.DefaultDueDays)

	fmt.Printf("\nFaktura: %s\n", invoice.InvoiceNumber)
	fmt.Printf("Odběratel: %s\n", customer.Name)
	fmt.Printf("Splatnost: %s\n", invoice.DueDate.Format("02.01.2006"))
	fmt.Println()

	// Add items
	var items []model.InvoiceItem
	position := 0

	for {
		fmt.Println("Přidat položku:")
		fmt.Println("  N) Nová položka")
		fmt.Println("  D) Hotovo")
		fmt.Println("  0) Zrušit fakturu")
		fmt.Println()

		choice := c.prompt("Volba")

		if choice == "0" {
			if c.confirm("Opravdu zrušit vytváření faktury?") {
				return
			}
			continue
		}

		if choice == "d" || choice == "D" || choice == "" {
			break
		}

		if choice == "n" || choice == "N" {
			item, goBack := c.addInvoiceItemWithBack(invoice.ID, position)
			if goBack {
				continue
			}
			if item != nil {
				items = append(items, *item)
				position++

				// Show current total
				var total float64
				for _, it := range items {
					total += it.Total
				}
				fmt.Printf("\n  Aktuální součet: %.2f %s\n\n", total, invoice.Currency)
			}
		}
	}

	if len(items) == 0 {
		c.printError("Faktura musí mít alespoň jednu položku")
		c.waitEnter()
		return
	}

	// Calculate totals
	for _, item := range items {
		invoice.Subtotal += item.Subtotal
		invoice.VATTotal += item.VATAmount
		invoice.Total += item.Total
	}

	// Show summary
	c.clearScreen()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                    SOUHRN FAKTURY")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Číslo faktury: %s\n", invoice.InvoiceNumber)
	fmt.Printf("Odběratel:     %s\n", customer.Name)
	fmt.Printf("Datum:         %s\n", invoice.IssueDate.Format("02.01.2006"))
	fmt.Printf("Splatnost:     %s\n", invoice.DueDate.Format("02.01.2006"))
	fmt.Println()
	fmt.Println("Položky:")
	for _, item := range items {
		fmt.Printf("  %.0fx %s @ %.2f %s = %.2f %s\n",
			item.Quantity, item.Description, item.UnitPrice,
			invoice.Currency, item.Total, invoice.Currency)
	}
	fmt.Println()
	fmt.Printf("                              Základ:  %10.2f %s\n", invoice.Subtotal, invoice.Currency)
	fmt.Printf("                              DPH:     %10.2f %s\n", invoice.VATTotal, invoice.Currency)
	fmt.Println("                              ─────────────────────")
	fmt.Printf("                              CELKEM:  %10.2f %s\n", invoice.Total, invoice.Currency)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("  U) Uložit fakturu")
	fmt.Println("  Z) Zrušit")
	fmt.Println()

	choice := c.prompt("Volba")

	if choice != "u" && choice != "U" {
		fmt.Println("Faktura nebyla uložena.")
		c.waitEnter()
		return
	}

	// Save invoice
	if err := c.invoices.Create(invoice); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Save items
	for i := range items {
		items[i].InvoiceID = invoice.ID
	}
	if err := c.invItems.CreateBatch(items); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(fmt.Sprintf("Faktura %s byla vytvořena!", invoice.InvoiceNumber))

	// Ask to generate PDF
	if c.confirm("Generovat PDF?") {
		c.generatePDF(invoice)
	}

	// Ask to change status
	if c.confirm("Označit jako odeslanou?") {
		c.invoices.UpdateStatus(invoice.ID, model.StatusSent)
		invoice.Status = model.StatusSent
	}

	c.waitEnter()
}

func (c *CLI) selectSupplierForInvoice() (*model.Supplier, bool) {
	suppliers, err := c.suppliers.List()
	if err != nil || len(suppliers) == 0 {
		c.printError("Není nastaven žádný dodavatel")
		c.waitEnter()
		return nil, true
	}

	// If only one supplier, use it automatically
	if len(suppliers) == 1 {
		fmt.Printf("Dodavatel: %s\n", suppliers[0].Name)
		return suppliers[0], false
	}

	// Find default
	var defaultIdx int
	for i, s := range suppliers {
		if s.IsDefault {
			defaultIdx = i
			break
		}
	}

	fmt.Println("Vyberte dodavatele (vaši firmu):")
	for i, s := range suppliers {
		def := ""
		if s.IsDefault {
			def = " [výchozí]"
		}
		fmt.Printf("  %d) %s%s\n", i+1, s.Name, def)
	}
	fmt.Println("  0) Zpět")
	fmt.Println()

	choice := c.promptDefault("Volba", fmt.Sprintf("%d", defaultIdx+1))

	if choice == "0" {
		return nil, true
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(suppliers) {
		fmt.Printf("Dodavatel: %s\n", suppliers[idx-1].Name)
		return suppliers[idx-1], false
	}

	// Use default if invalid input
	return suppliers[defaultIdx], false
}

func (c *CLI) selectBankAccountForInvoice(supplierID string) (*model.BankAccount, bool) {
	accounts, err := c.bankAccs.GetBySupplier(supplierID)
	if err != nil || len(accounts) == 0 {
		c.printError("Není nastaven žádný bankovní účet")
		c.waitEnter()
		return nil, true
	}

	// If only one account, use it automatically
	if len(accounts) == 1 {
		fmt.Printf("Účet: %s (%s)\n", accounts[0].AccountNumber, accounts[0].Currency)
		return accounts[0], false
	}

	// Find default
	var defaultIdx int
	for i, a := range accounts {
		if a.IsDefault {
			defaultIdx = i
			break
		}
	}

	fmt.Println()
	fmt.Println("Vyberte bankovní účet:")
	for i, a := range accounts {
		def := ""
		if a.IsDefault {
			def = " [výchozí]"
		}
		name := a.Name
		if name == "" {
			name = a.Currency
		}
		fmt.Printf("  %d) %s - %s%s\n", i+1, name, a.AccountNumber, def)
	}
	fmt.Println("  0) Zpět")
	fmt.Println()

	choice := c.promptDefault("Volba", fmt.Sprintf("%d", defaultIdx+1))

	if choice == "0" {
		return nil, true
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(accounts) {
		fmt.Printf("Účet: %s (%s)\n", accounts[idx-1].AccountNumber, accounts[idx-1].Currency)
		return accounts[idx-1], false
	}

	// Use default if invalid input
	return accounts[defaultIdx], false
}

func (c *CLI) addInvoiceItem(invoiceID string, position int) *model.InvoiceItem {
	item, _ := c.addInvoiceItemWithBack(invoiceID, position)
	return item
}

func (c *CLI) addInvoiceItemWithBack(invoiceID string, position int) (*model.InvoiceItem, bool) {
	item := model.NewInvoiceItem(invoiceID)
	item.Position = position

	desc, goBack := c.promptWithBack("Popis položky")
	if goBack {
		return nil, true
	}
	if desc == "" {
		return nil, false
	}
	item.Description = desc

	item.Quantity = c.promptFloat("Množství", 1)
	item.Unit = c.promptDefault("Jednotka", "ks")
	item.UnitPrice = c.promptFloat("Cena za jednotku", 0)
	item.VATRate = c.promptFloat("DPH %", 0)

	item.Calculate()

	fmt.Printf("  → %.0f %s × %.2f = %.2f (DPH: %.2f)\n",
		item.Quantity, item.Unit, item.UnitPrice, item.Total, item.VATAmount)

	return item, false
}

func (c *CLI) listInvoices() {
	c.clearScreen()
	fmt.Println("=== SEZNAM FAKTUR ===")
	fmt.Println()

	invoices, err := c.invoices.List("", "")
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(invoices) == 0 {
		fmt.Println("Žádné faktury.")
		c.waitEnter()
		return
	}

	for i, inv := range invoices {
		customer, _ := c.customers.GetByID(inv.CustomerID)
		custName := "?"
		if customer != nil {
			custName = customer.Name
		}

		statusIcon := c.statusIcon(inv.Status)
		fmt.Printf("  %d) %s %s | %s | %-20s | %10.2f %s\n",
			i+1, statusIcon, inv.InvoiceNumber,
			inv.IssueDate.Format("02.01.2006"),
			custName, inv.Total, inv.Currency)
	}

	fmt.Println()
	fmt.Println("  0) Zpět")
	fmt.Println()

	choice := c.prompt("Vyberte fakturu pro detail")
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(invoices) {
		c.invoiceDetail(invoices[idx-1])
	}
}

func (c *CLI) listUnpaidInvoices() {
	c.clearScreen()
	fmt.Println("=== NEZAPLACENÉ FAKTURY ===")
	fmt.Println()

	invoices, err := c.invoices.ListUnpaid()
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(invoices) == 0 {
		fmt.Println("Všechny faktury jsou zaplaceny!")
		c.waitEnter()
		return
	}

	now := time.Now()
	for i, inv := range invoices {
		customer, _ := c.customers.GetByID(inv.CustomerID)
		custName := "?"
		if customer != nil {
			custName = customer.Name
		}

		overdue := ""
		if inv.DueDate.Before(now) {
			days := int(now.Sub(inv.DueDate).Hours() / 24)
			overdue = fmt.Sprintf(" [PO SPLATNOSTI %d dní]", days)
		}

		fmt.Printf("  %d) %s | splatnost %s | %-15s | %10.2f %s%s\n",
			i+1, inv.InvoiceNumber,
			inv.DueDate.Format("02.01.2006"),
			custName, inv.Total, inv.Currency, overdue)
	}

	fmt.Println()
	fmt.Println("  0) Zpět")
	fmt.Println()

	choice := c.prompt("Vyberte fakturu")
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(invoices) {
		c.invoiceDetail(invoices[idx-1])
	}
}

func (c *CLI) invoiceDetail(inv *model.Invoice) {
	for {
		c.clearScreen()

		customer, _ := c.customers.GetByID(inv.CustomerID)
		custName := "?"
		if customer != nil {
			custName = customer.Name
		}

		items, _ := c.invItems.GetByInvoice(inv.ID)

		fmt.Printf("=== FAKTURA %s ===\n", inv.InvoiceNumber)
		fmt.Println()
		fmt.Printf("  Stav:        %s %s\n", c.statusIcon(inv.Status), c.statusName(inv.Status))
		fmt.Printf("  Odběratel:   %s\n", custName)
		fmt.Printf("  Vystaveno:   %s\n", inv.IssueDate.Format("02.01.2006"))
		fmt.Printf("  Splatnost:   %s\n", inv.DueDate.Format("02.01.2006"))
		fmt.Printf("  VS:          %s\n", inv.VariableSymbol)
		fmt.Println()

		fmt.Println("  Položky:")
		for _, item := range items {
			fmt.Printf("    %.0f× %s @ %.2f = %.2f %s\n",
				item.Quantity, item.Description, item.UnitPrice, item.Total, inv.Currency)
		}
		fmt.Println()
		fmt.Printf("  CELKEM: %.2f %s\n", inv.Total, inv.Currency)
		fmt.Println()

		fmt.Println("  G) Generovat PDF")
		if inv.PDFPath != "" {
			fmt.Println("  O) Otevřít PDF")
		}
		fmt.Println("  S) Změnit stav")
		fmt.Println("  P) Označit jako zaplacenou")
		fmt.Println("  X) Smazat fakturu")
		fmt.Println("  0) Zpět")
		fmt.Println()

		choice := c.prompt("Volba")

		switch choice {
		case "0", "":
			return
		case "g", "G":
			c.generatePDF(inv)
		case "o", "O":
			if inv.PDFPath != "" {
				c.openFile(inv.PDFPath)
			}
		case "s", "S":
			c.changeInvoiceStatus(inv)
		case "p", "P":
			c.invoices.UpdateStatus(inv.ID, model.StatusPaid)
			inv.Status = model.StatusPaid
			c.printSuccess("Faktura označena jako zaplacená")
		case "x", "X":
			if c.confirm("Opravdu smazat fakturu?") {
				c.invoices.Delete(inv.ID)
				c.printSuccess("Faktura smazána")
				c.waitEnter()
				return
			}
		}
	}
}

func (c *CLI) generatePDF(inv *model.Invoice) {
	supplier, err := c.suppliers.GetByID(inv.SupplierID)
	if err != nil || supplier == nil {
		c.printError("Nepodařilo se načíst dodavatele")
		return
	}

	customer, err := c.customers.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		c.printError("Nepodařilo se načíst odběratele")
		return
	}

	bankAcc, err := c.bankAccs.GetByID(inv.BankAccountID)
	if err != nil || bankAcc == nil {
		c.printError("Nepodařilo se načíst bankovní účet")
		return
	}

	items, err := c.invItems.GetByInvoice(inv.ID)
	if err != nil {
		c.printError("Nepodařilo se načíst položky")
		return
	}

	fmt.Println("Generuji PDF...")

	data := &service.InvoiceData{
		Invoice:     inv,
		Supplier:    supplier,
		Customer:    customer,
		BankAccount: bankAcc,
		Items:       items,
	}

	pdfPath, err := c.pdfService.GenerateInvoice(data)
	if err != nil {
		c.printError(err.Error())
		return
	}

	// Update invoice with PDF path
	inv.PDFPath = pdfPath
	c.invoices.Update(inv)

	c.printSuccess(fmt.Sprintf("PDF vytvořeno: %s", pdfPath))

	if c.confirm("Otevřít PDF?") {
		c.openFile(pdfPath)
	}
}

func (c *CLI) openFile(path string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		c.printError(fmt.Sprintf("Nepodařilo se otevřít soubor: %v", err))
	}
}

func (c *CLI) changeInvoiceStatus(inv *model.Invoice) {
	fmt.Println()
	fmt.Println("Vyberte nový stav:")
	fmt.Println("  1) Koncept")
	fmt.Println("  2) Vytvořena")
	fmt.Println("  3) Odeslaná")
	fmt.Println("  4) Zaplacená")
	fmt.Println("  5) Po splatnosti")
	fmt.Println("  6) Zrušená")
	fmt.Println()

	choice := c.prompt("Volba")

	var newStatus model.InvoiceStatus
	switch choice {
	case "1":
		newStatus = model.StatusDraft
	case "2":
		newStatus = model.StatusCreated
	case "3":
		newStatus = model.StatusSent
	case "4":
		newStatus = model.StatusPaid
	case "5":
		newStatus = model.StatusOverdue
	case "6":
		newStatus = model.StatusCancelled
	default:
		return
	}

	c.invoices.UpdateStatus(inv.ID, newStatus)
	inv.Status = newStatus
	c.printSuccess("Stav změněn")
}

func (c *CLI) createFromExisting() {
	// TODO: implement duplicate invoice
	fmt.Println("Funkce bude brzy dostupná...")
	c.waitEnter()
}

func (c *CLI) statusIcon(status model.InvoiceStatus) string {
	switch status {
	case model.StatusDraft:
		return "📝"
	case model.StatusCreated:
		return "📄"
	case model.StatusSent:
		return "📤"
	case model.StatusPaid:
		return "✅"
	case model.StatusOverdue:
		return "⚠️"
	case model.StatusCancelled:
		return "❌"
	default:
		return "❓"
	}
}

func (c *CLI) statusName(status model.InvoiceStatus) string {
	switch status {
	case model.StatusDraft:
		return "Koncept"
	case model.StatusCreated:
		return "Vytvořena"
	case model.StatusSent:
		return "Odeslaná"
	case model.StatusPaid:
		return "Zaplacená"
	case model.StatusOverdue:
		return "Po splatnosti"
	case model.StatusCancelled:
		return "Zrušená"
	default:
		return string(status)
	}
}
