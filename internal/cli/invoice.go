package cli

import (
	"database/sql"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
	"github.com/adamSHA256/tidybill/internal/service"
)

func (c *CLI) createInvoice() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.new_invoice"))
	fmt.Println(i18n.T("prompt.enter_0_back_anytime"))
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

	fmt.Println()
	fmt.Println(i18n.Tf("label.invoice_number", invoice.InvoiceNumber))
	fmt.Println(i18n.Tf("label.customer_short", customer.Name))
	fmt.Println(i18n.Tf("label.due_date_short", invoice.DueDate.Format("02.01.2006")))
	fmt.Println()

	// Add items
	var items []model.InvoiceItem
	position := 0

	for {
		fmt.Println(i18n.T("prompt.add_item"))
		fmt.Println("  " + i18n.T("action.new_item"))
		fmt.Println("  " + i18n.T("action.from_catalog"))
		fmt.Println("  " + i18n.T("action.done"))
		fmt.Println("  " + i18n.T("action.cancel_invoice"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choice"))

		if choice == "0" {
			if c.confirm(i18n.T("confirm.cancel_invoice")) {
				return
			}
			continue
		}

		if choice == "d" || choice == "D" || choice == "" {
			break
		}

		if choice == "n" || choice == "N" {
			item, goBack := c.addInvoiceItemWithBack(invoice.ID, position, nil)
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
				fmt.Printf("\n  %s\n\n", i18n.Tf("label.current_total", total, invoice.Currency))
			}
		}

		if strings.ToLower(choice) == "k" {
			catalogItem := c.selectFromCatalog(customer.ID)
			if catalogItem != nil {
				item, goBack := c.addInvoiceItemWithBack(invoice.ID, position, catalogItem)
				if !goBack && item != nil {
					items = append(items, *item)
					position++

					var total float64
					for _, it := range items {
						total += it.Total
					}
					fmt.Printf("\n  %s\n\n", i18n.Tf("label.current_total", total, invoice.Currency))
				}
			}
		}
	}

	if len(items) == 0 {
		c.printError(i18n.T("error.invoice_no_items"))
		c.waitEnter()
		return
	}

	// Invoice notes (printed on PDF)
	invoice.Notes = c.prompt(i18n.T("prompt.invoice_notes"))

	// Calculate totals
	for _, item := range items {
		invoice.Subtotal += item.Subtotal
		invoice.VATTotal += item.VATAmount
		invoice.Total += item.Total
	}

	// Show summary
	c.clearScreen()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("                    %s\n", i18n.T("heading.invoice_summary"))
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println(i18n.Tf("label.invoice_number_full", invoice.InvoiceNumber))
	fmt.Println(i18n.Tf("label.customer", customer.Name))
	fmt.Println(i18n.Tf("label.date", invoice.IssueDate.Format("02.01.2006")))
	fmt.Println(i18n.Tf("label.due_date", invoice.DueDate.Format("02.01.2006")))
	fmt.Println()
	fmt.Println(i18n.T("label.items"))
	for _, item := range items {
		fmt.Printf("  %.0fx %s @ %.2f %s = %.2f %s\n",
			item.Quantity, item.Description, item.UnitPrice,
			invoice.Currency, item.Total, invoice.Currency)
	}
	fmt.Println()
	fmt.Printf("                              %-10s %10.2f %s\n", i18n.T("label.subtotal"), invoice.Subtotal, invoice.Currency)
	fmt.Printf("                              %-10s %10.2f %s\n", i18n.T("label.vat"), invoice.VATTotal, invoice.Currency)
	fmt.Println("                              ─────────────────────")
	fmt.Printf("                              %-10s %10.2f %s\n", i18n.T("label.total"), invoice.Total, invoice.Currency)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("  " + i18n.T("action.save_invoice"))
	fmt.Println("  " + i18n.T("action.cancel"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.choice"))

	if choice != "u" && choice != "U" {
		fmt.Println(i18n.T("info.invoice_not_saved"))
		c.waitEnter()
		return
	}

	// Save invoice + items + usage tracking atomically
	if err := repository.WithTx(c.db.DB, func(tx *sql.Tx) error {
		if err := c.invoices.WithDB(tx).Create(invoice); err != nil {
			return fmt.Errorf("save invoice: %w", err)
		}

		for i := range items {
			items[i].InvoiceID = invoice.ID
		}
		if err := c.invItems.WithDB(tx).CreateBatch(items); err != nil {
			return fmt.Errorf("save items: %w", err)
		}

		itemRepo := c.items.WithDB(tx)
		custItemRepo := c.custItems.WithDB(tx)
		for _, invItem := range items {
			if invItem.ItemID == "" {
				continue
			}
			if err := itemRepo.IncrementUsage(invItem.ItemID, invItem.UnitPrice, customer.ID); err != nil {
				return fmt.Errorf("update item usage: %w", err)
			}
			if err := custItemRepo.Upsert(customer.ID, invItem.ItemID, invItem.UnitPrice, invItem.Quantity); err != nil {
				return fmt.Errorf("update customer item: %w", err)
			}
		}

		return nil
	}); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(i18n.Tf("success.invoice_created", invoice.InvoiceNumber))

	// Ask to generate PDF
	if c.confirm(i18n.T("confirm.generate_pdf")) {
		c.generatePDF(invoice)
	}

	// Ask to change status
	if c.confirm(i18n.T("confirm.mark_as_sent")) {
		c.invoices.UpdateStatus(invoice.ID, model.StatusSent)
		invoice.Status = model.StatusSent
	}

	c.waitEnter()
}

func (c *CLI) selectSupplierForInvoice() (*model.Supplier, bool) {
	suppliers, err := c.suppliers.List()
	if err != nil || len(suppliers) == 0 {
		c.printError(i18n.T("error.no_supplier"))
		c.waitEnter()
		return nil, true
	}

	// If only one supplier, use it automatically
	if len(suppliers) == 1 {
		fmt.Println(i18n.Tf("label.supplier", suppliers[0].Name))
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

	fmt.Println(i18n.T("prompt.select_supplier"))
	for i, s := range suppliers {
		def := ""
		if s.IsDefault {
			def = " " + i18n.T("label.default_lower")
		}
		fmt.Printf("  %d) %s%s\n", i+1, s.Name, def)
	}
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.promptDefault(i18n.T("prompt.choice"), fmt.Sprintf("%d", defaultIdx+1))

	if choice == "0" {
		return nil, true
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(suppliers) {
		fmt.Println(i18n.Tf("label.supplier", suppliers[idx-1].Name))
		return suppliers[idx-1], false
	}

	// Use default if invalid input
	return suppliers[defaultIdx], false
}

func (c *CLI) selectBankAccountForInvoice(supplierID string) (*model.BankAccount, bool) {
	accounts, err := c.bankAccs.GetBySupplier(supplierID)
	if err != nil || len(accounts) == 0 {
		c.printError(i18n.T("error.no_bank_account"))
		c.waitEnter()
		return nil, true
	}

	// If only one account, use it automatically
	if len(accounts) == 1 {
		fmt.Println(i18n.Tf("label.account_with_currency", accounts[0].AccountNumber, accounts[0].Currency))
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
	fmt.Println(i18n.T("prompt.select_bank_account"))
	for i, a := range accounts {
		def := ""
		if a.IsDefault {
			def = " " + i18n.T("label.default_lower")
		}
		name := a.Name
		if name == "" {
			name = a.Currency
		}
		fmt.Printf("  %d) %s - %s%s\n", i+1, name, a.AccountNumber, def)
	}
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.promptDefault(i18n.T("prompt.choice"), fmt.Sprintf("%d", defaultIdx+1))

	if choice == "0" {
		return nil, true
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(accounts) {
		fmt.Println(i18n.Tf("label.account_with_currency", accounts[idx-1].AccountNumber, accounts[idx-1].Currency))
		return accounts[idx-1], false
	}

	// Use default if invalid input
	return accounts[defaultIdx], false
}

func (c *CLI) addInvoiceItem(invoiceID string, position int) *model.InvoiceItem {
	item, _ := c.addInvoiceItemWithBack(invoiceID, position, nil)
	return item
}

func (c *CLI) addInvoiceItemWithBack(invoiceID string, position int, catalogItem *model.Item) (*model.InvoiceItem, bool) {
	item := model.NewInvoiceItem(invoiceID)
	item.Position = position

	var defaultDesc, defaultUnit string
	var defaultPrice, defaultVAT float64
	if catalogItem != nil {
		defaultDesc = catalogItem.Description
		defaultUnit = catalogItem.DefaultUnit
		defaultPrice = catalogItem.DefaultPrice
		defaultVAT = catalogItem.DefaultVATRate
		item.ItemID = catalogItem.ID

		fmt.Printf("  %s: %s\n", i18n.T("label.from_catalog"), catalogItem.Description)
	}

	var desc string
	var goBack bool
	if defaultDesc != "" {
		desc = c.promptDefaultMaxLen(i18n.T("prompt.item_description"), defaultDesc, model.MaxDescriptionLen)
	} else {
		desc, goBack = c.promptMaxLenWithBack(i18n.T("prompt.item_description"), model.MaxDescriptionLen)
		if goBack {
			return nil, true
		}
	}
	if desc == "" {
		return nil, false
	}
	item.Description = desc

	item.Quantity = c.promptFloat(i18n.T("prompt.quantity"), 1)
	if defaultUnit != "" {
		item.Unit = c.promptDefault(i18n.T("prompt.unit"), defaultUnit)
	} else {
		item.Unit = c.promptDefault(i18n.T("prompt.unit"), i18n.T("default.unit_pcs"))
	}
	item.UnitPrice = c.promptFloat(i18n.T("prompt.unit_price"), defaultPrice)
	item.VATRate = c.promptFloat(i18n.T("prompt.vat_rate"), defaultVAT)

	item.Calculate()

	fmt.Printf("  → %.0f %s × %.2f = %.2f (DPH: %.2f)\n",
		item.Quantity, item.Unit, item.UnitPrice, item.Total, item.VATAmount)

	return item, false
}

func (c *CLI) listInvoices() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.invoice_list"))
	fmt.Println()

	invoices, err := c.invoices.List("", "")
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(invoices) == 0 {
		fmt.Println(i18n.T("info.no_invoices"))
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
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.select_invoice"))
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(invoices) {
		c.invoiceDetail(invoices[idx-1])
	}
}

func (c *CLI) listUnpaidInvoices() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.unpaid_invoices"))
	fmt.Println()

	invoices, err := c.invoices.ListUnpaid()
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(invoices) == 0 {
		fmt.Println(i18n.T("info.all_paid"))
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
			overdue = fmt.Sprintf(" [%s]", i18n.Tf("label.overdue_days", days))
		}

		fmt.Printf("  %d) %s | %s %s | %-15s | %10.2f %s%s\n",
			i+1, inv.InvoiceNumber,
			i18n.T("label.due"), inv.DueDate.Format("02.01.2006"),
			custName, inv.Total, inv.Currency, overdue)
	}

	fmt.Println()
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.select_invoice_short"))
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

		fmt.Printf("=== %s ===\n", i18n.Tf("heading.invoice_detail", inv.InvoiceNumber))
		fmt.Println()
		fmt.Printf("  "+i18n.T("label.status")+"\n", c.statusIcon(inv.Status), c.statusName(inv.Status))
		fmt.Printf("  "+i18n.T("label.customer")+"\n", custName)
		fmt.Printf("  "+i18n.T("label.issued")+"\n", inv.IssueDate.Format("02.01.2006"))
		fmt.Printf("  "+i18n.T("label.due_date")+"\n", inv.DueDate.Format("02.01.2006"))
		fmt.Printf("  "+i18n.T("label.variable_symbol")+"\n", inv.VariableSymbol)
		fmt.Println()

		if inv.Notes != "" {
			c.printMultiline("  ", i18n.T("label.notes"), inv.Notes)
		}
		if inv.InternalNotes != "" {
			c.printMultiline("  ", i18n.T("label.internal_notes"), inv.InternalNotes)
		}
		fmt.Println()

		fmt.Println("  " + i18n.T("label.items"))
		for _, item := range items {
			fmt.Printf("    %.0f× %s @ %.2f = %.2f %s\n",
				item.Quantity, item.Description, item.UnitPrice, item.Total, inv.Currency)
		}
		fmt.Println()
		fmt.Printf("  %s %.2f %s\n", i18n.T("label.total"), inv.Total, inv.Currency)
		fmt.Println()

		fmt.Println("  " + i18n.T("action.generate_pdf"))
		if inv.PDFPath != "" {
			fmt.Println("  " + i18n.T("action.open_pdf"))
		}
		fmt.Println("  " + i18n.T("action.internal_notes"))
		fmt.Println("  " + i18n.T("action.change_status"))
		fmt.Println("  " + i18n.T("action.mark_paid"))
		fmt.Println("  " + i18n.T("action.delete_invoice"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choice"))

		switch choice {
		case "0", "":
			return
		case "n", "N":
			c.editInvoiceInternalNotes(inv)
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
			c.printSuccess(i18n.T("success.invoice_paid"))
		case "x", "X":
			if c.confirm(i18n.T("confirm.delete_invoice")) {
				c.invoices.Delete(inv.ID)
				c.printSuccess(i18n.T("success.invoice_deleted"))
				c.waitEnter()
				return
			}
		}
	}
}

func (c *CLI) generatePDF(inv *model.Invoice) {
	supplier, err := c.suppliers.GetByID(inv.SupplierID)
	if err != nil || supplier == nil {
		c.printError(i18n.T("error.load_supplier"))
		return
	}

	customer, err := c.customers.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		c.printError(i18n.T("error.load_customer"))
		return
	}

	bankAcc, err := c.bankAccs.GetByID(inv.BankAccountID)
	if err != nil || bankAcc == nil {
		c.printError(i18n.T("error.load_bank_account"))
		return
	}

	items, err := c.invItems.GetByInvoice(inv.ID)
	if err != nil {
		c.printError(i18n.T("error.load_items"))
		return
	}

	fmt.Println(i18n.T("info.generating_pdf"))

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

	c.printSuccess(i18n.Tf("success.pdf_created", pdfPath))

	if c.confirm(i18n.T("confirm.open_pdf")) {
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
		c.printError(i18n.Tf("error.open_file", err))
	}
}

func (c *CLI) changeInvoiceStatus(inv *model.Invoice) {
	fmt.Println()
	fmt.Println(i18n.T("prompt.select_status"))
	fmt.Printf("  1) %s\n", i18n.T("status.draft"))
	fmt.Printf("  2) %s\n", i18n.T("status.created"))
	fmt.Printf("  3) %s\n", i18n.T("status.sent"))
	fmt.Printf("  4) %s\n", i18n.T("status.paid"))
	fmt.Printf("  5) %s\n", i18n.T("status.overdue"))
	fmt.Printf("  6) %s\n", i18n.T("status.cancelled"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.choice"))

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
	c.printSuccess(i18n.T("success.status_changed"))
}

func (c *CLI) editInvoiceInternalNotes(inv *model.Invoice) {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.Tf("heading.invoice_detail", inv.InvoiceNumber))
	fmt.Printf("\n  %s\n\n", i18n.T("action.internal_notes"))

	inv.InternalNotes = c.editNotes(inv.InternalNotes)

	if err := c.invoices.Update(inv); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.notes_saved"))
	}
	c.waitEnter()
}

func (c *CLI) createFromExisting() {
	// TODO: implement duplicate invoice
	fmt.Println(i18n.T("info.coming_soon"))
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
		return i18n.T("status.draft")
	case model.StatusCreated:
		return i18n.T("status.created")
	case model.StatusSent:
		return i18n.T("status.sent")
	case model.StatusPaid:
		return i18n.T("status.paid")
	case model.StatusOverdue:
		return i18n.T("status.overdue")
	case model.StatusCancelled:
		return i18n.T("status.cancelled")
	default:
		return string(status)
	}
}

func (c *CLI) selectFromCatalog(customerID string) *model.Item {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.select_from_catalog"))
	fmt.Println()

	customerItems, _ := c.custItems.GetByCustomer(customerID)
	globalItems, _ := c.items.GetMostUsed(10)

	type catalogEntry struct {
		item      *model.Item
		custPrice float64
		custCount int
	}

	var entries []catalogEntry
	seen := make(map[string]bool)

	if len(customerItems) > 0 {
		fmt.Println("  " + i18n.T("label.customer_items"))
		for _, ci := range customerItems {
			fullItem, _ := c.items.GetByID(ci.ItemID)
			if fullItem == nil {
				continue
			}
			seen[ci.ItemID] = true
			entries = append(entries, catalogEntry{
				item:      fullItem,
				custPrice: ci.LastPrice,
				custCount: ci.UsageCount,
			})
		}
	}

	var globalEntries []catalogEntry
	for _, item := range globalItems {
		if seen[item.ID] {
			continue
		}
		globalEntries = append(globalEntries, catalogEntry{item: item})
	}

	if len(globalEntries) > 0 {
		if len(customerItems) > 0 {
			fmt.Println()
		}
		fmt.Println("  " + i18n.T("label.global_items"))
		entries = append(entries, globalEntries...)
	}

	if len(entries) == 0 {
		fmt.Println("  " + i18n.T("info.no_catalog_items"))
		fmt.Println("  " + i18n.T("info.create_item_first"))
		fmt.Println()

		if c.confirm(i18n.T("prompt.create_new_item_yn")) {
			return c.createItem()
		}
		return nil
	}

	for i, entry := range entries {
		priceInfo := fmt.Sprintf("%.2f", entry.item.DefaultPrice)
		if entry.custPrice > 0 && entry.custPrice != entry.item.DefaultPrice {
			priceInfo = fmt.Sprintf("%.2f (%s: %.2f)",
				entry.custPrice, i18n.T("label.catalog_price"), entry.item.DefaultPrice)
		} else if entry.custPrice > 0 {
			priceInfo = fmt.Sprintf("%.2f", entry.custPrice)
		}

		usageInfo := ""
		if entry.custCount > 0 {
			usageInfo = fmt.Sprintf(" [%dx]", entry.custCount)
		} else if entry.item.UsageCount > 0 {
			usageInfo = fmt.Sprintf(" [%dx %s]", entry.item.UsageCount, i18n.T("label.global"))
		}

		fmt.Printf("  %d) %s — %s %s%s\n",
			i+1, entry.item.Description, priceInfo, entry.item.DefaultUnit, usageInfo)
	}

	fmt.Println()
	fmt.Println("  " + i18n.T("action.new_item_inline"))
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.choice"))

	if choice == "0" || choice == "" {
		return nil
	}

	if strings.ToLower(choice) == "n" {
		return c.createItem()
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(entries) {
		entry := entries[idx-1]
		if entry.custPrice > 0 {
			entry.item.DefaultPrice = entry.custPrice
		}
		return entry.item
	}

	return nil
}
