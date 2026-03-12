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

// saveInvoiceWithSummary displays invoice summary, prompts for save, and saves atomically.
// Returns true if saved successfully.
func (c *CLI) saveInvoiceWithSummary(invoice *model.Invoice, items []model.InvoiceItem, customer *model.Customer) bool {
	// Calculate totals
	invoice.Subtotal = 0
	invoice.VATTotal = 0
	invoice.Total = 0
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

	choice := c.promptDefault(i18n.T("prompt.choice"), "u")

	if choice != "u" && choice != "U" {
		fmt.Println(i18n.T("info.invoice_not_saved"))
		c.waitEnter()
		return false
	}

	// Save invoice + items + usage tracking atomically
	if err := repository.WithTx(c.db.DB, func(tx *sql.Tx) error {
		if err := c.invoices.WithDB(tx).Create(invoice); err != nil {
			return fmt.Errorf("save invoice: %w", err)
		}

		// Auto-create catalog entries for manually-added items
		itemRepo := c.items.WithDB(tx)
		for i := range items {
			items[i].InvoiceID = invoice.ID
			if items[i].ItemID == "" {
				existing, _ := itemRepo.FindByDescription(items[i].Description)
				if existing != nil {
					items[i].ItemID = existing.ID
				} else {
					catalogItem := &model.Item{
						Description:    items[i].Description,
						DefaultPrice:   items[i].UnitPrice,
						DefaultUnit:    items[i].Unit,
						DefaultVATRate: items[i].VATRate,
						LastUsedPrice:  items[i].UnitPrice,
						LastCustomerID: customer.ID,
						UsageCount:     0,
					}
					if err := itemRepo.Create(catalogItem); err != nil {
						return fmt.Errorf("create catalog item: %w", err)
					}
					items[i].ItemID = catalogItem.ID
				}
			}
		}

		if err := c.invItems.WithDB(tx).CreateBatch(items); err != nil {
			return fmt.Errorf("save items: %w", err)
		}

		// Track usage for all items
		custItemRepo := c.custItems.WithDB(tx)
		for _, invItem := range items {
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
		return false
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
	return true
}

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

	// Generate invoice number (user can override)
	invNumber, err := c.invoices.GetNextNumber(supplier.ID, supplier.InvoicePrefix)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}
	invNumber = c.promptDefault(i18n.T("prompt.invoice_number"), invNumber)

	// Select payment method + bank account (looped so user can go back to change payment method)
	var selectedPaymentType cliPaymentType
	var bankAcc *model.BankAccount
	var requiresBankInfo bool
	for {
		selectedPaymentType = c.selectPaymentTypeStruct()
		requiresBankInfo = selectedPaymentType.RequiresBankInfo == nil || *selectedPaymentType.RequiresBankInfo
		if requiresBankInfo {
			var goBack bool
			bankAcc, goBack = c.selectBankAccountForInvoice(supplier.ID)
			if goBack {
				continue // back to payment method selection
			}
		} else {
			bankAcc = nil
		}
		break
	}

	bankAccountID := ""
	if bankAcc != nil {
		bankAccountID = bankAcc.ID
	}

	// Create invoice
	invoice := model.NewInvoice(supplier.ID, customer.ID, bankAccountID)
	invoice.InvoiceNumber = invNumber
	invoice.PaymentMethod = selectedPaymentType.Name

	if requiresBankInfo {
		invoice.VariableSymbol = repository.GenerateVariableSymbol(invNumber)
		// Allow user to override VS
		invoice.VariableSymbol = c.promptDefault(i18n.T("prompt.variable_symbol"), invoice.VariableSymbol)
		invoice.Currency = bankAcc.Currency
	} else {
		invoice.VariableSymbol = ""
		// Prefer supplier's default bank account currency, fall back to global default
		if defBank, err := c.bankAccs.GetDefaultForSupplier(supplier.ID); err == nil && defBank != nil {
			invoice.Currency = defBank.Currency
		} else {
			invoice.Currency = c.getDefaultCurrency()
		}
	}

	// Due date: use customer default, fall back to global setting
	dueDays := customer.DefaultDueDays
	if dueDays == 0 {
		dueDays = c.getDefaultDueDaysInt()
	}
	invoice.DueDate = time.Now().AddDate(0, 0, dueDays)

	fmt.Println()
	fmt.Println(i18n.Tf("label.invoice_number", invoice.InvoiceNumber))
	fmt.Println(i18n.Tf("label.customer_short", customer.Name))

	// Allow user to change currency
	defaultCurrency := invoice.Currency
	invoice.Currency = c.promptDefault(i18n.T("prompt.currency"), defaultCurrency)

	// Allow user to change issue date (default: today)
	issueDateStr := c.promptDefault(i18n.T("prompt.issue_date_confirm"), invoice.IssueDate.Format("02.01.2006"))
	if t, err := time.Parse("02.01.2006", issueDateStr); err == nil {
		invoice.IssueDate = t
		invoice.TaxableDate = t
	}

	// Allow user to change DUZP / taxable date (default: issue date)
	taxableDateStr := c.promptDefault(i18n.T("prompt.taxable_date_confirm"), invoice.TaxableDate.Format("02.01.2006"))
	if t, err := time.Parse("02.01.2006", taxableDateStr); err == nil {
		invoice.TaxableDate = t
	}

	// Allow user to change due date
	dueDateStr := c.promptDefault(i18n.T("prompt.due_date_confirm"), invoice.DueDate.Format("02.01.2006"))
	if t, err := time.Parse("02.01.2006", dueDateStr); err == nil {
		invoice.DueDate = t
	}
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

		if choice == "d" || choice == "D" {
			if len(items) == 0 {
				c.printError(i18n.T("error.invoice_no_items"))
				fmt.Println()
				continue
			}
			break
		}

		if choice == "" {
			if len(items) == 0 {
				c.printError(i18n.T("error.invoice_no_items"))
				fmt.Println()
				continue
			}
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

	c.saveInvoiceWithSummary(invoice, items, customer)
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
		fmt.Printf("\n⚠️  %s\n\n", i18n.T("error.no_bank_account"))
		fmt.Println("  1)", i18n.T("bank_account.add_new"))
		fmt.Println("  ", i18n.T("action.back"))
		choice := c.prompt(i18n.T("prompt.choice"))
		if choice == "1" {
			c.addBankAccount(supplierID)
			// Re-fetch accounts after adding
			accounts, err = c.bankAccs.GetBySupplier(supplierID)
			if err != nil || len(accounts) == 0 {
				return nil, true
			}
			// Use the newly created account (likely the only one or the default)
			for _, a := range accounts {
				if a.IsDefault {
					return a, false
				}
			}
			return accounts[0], false
		}
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
	item.Unit = c.selectUnit(defaultUnit)
	item.UnitPrice = c.promptFloat(i18n.T("prompt.unit_price"), defaultPrice)
	// TODO: Dynamic VAT rates from settings (GET /api/vat-rates) are not implemented in CLI.
	// The GUI version uses configurable VAT rate options with is_default selection (pill editor).
	// Implement manageVATRates() in CLI similar to manageUnits/managePaymentTypes.
	item.VATRate = c.promptFloat(i18n.T("prompt.vat_rate"), defaultVAT)

	item.Calculate()

	fmt.Printf("  → %.0f %s × %.2f = %.2f (DPH: %.2f)\n",
		item.Quantity, item.Unit, item.UnitPrice, item.Total, item.VATAmount)

	return item, false
}

func (c *CLI) listInvoices() {
	var filterStatus model.InvoiceStatus
	var filterCustomerID string
	var filterFrom, filterTo time.Time
	var filterLabel string

	for {
		c.clearScreen()

		header := i18n.T("heading.invoice_list")
		if filterLabel != "" {
			header += " " + i18n.Tf("label.active_filter", filterLabel)
		}
		fmt.Printf("=== %s ===\n", header)
		fmt.Println()

		invoices, err := c.invoices.ListFiltered(filterStatus, filterCustomerID, filterFrom, filterTo)
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		if len(invoices) == 0 {
			if filterLabel != "" {
				fmt.Println(i18n.T("info.no_invoices_filter"))
			} else {
				fmt.Println(i18n.T("info.no_invoices"))
			}
		} else {
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
		}

		fmt.Println()
		fmt.Println("  " + i18n.T("action.filter"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.select_invoice"))

		switch strings.ToLower(choice) {
		case "0", "":
			return
		case "f":
			c.invoiceFilterMenu(&filterStatus, &filterCustomerID, &filterFrom, &filterTo, &filterLabel)
		default:
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			if idx > 0 && idx <= len(invoices) {
				c.invoiceDetail(invoices[idx-1])
			}
		}
	}
}

func (c *CLI) invoiceFilterMenu(status *model.InvoiceStatus, customerID *string, from, to *time.Time, label *string) {
	fmt.Println()
	fmt.Println(i18n.T("prompt.filter_by"))
	fmt.Println("  " + i18n.T("action.filter_status"))
	fmt.Println("  " + i18n.T("action.filter_customer"))
	fmt.Println("  " + i18n.T("action.filter_date"))
	fmt.Println("  " + i18n.T("action.reset_filter"))
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.choice"))

	switch strings.ToLower(choice) {
	case "s":
		fmt.Println()
		fmt.Printf("  1) %s\n", i18n.T("status.draft"))
		fmt.Printf("  2) %s\n", i18n.T("status.created"))
		fmt.Printf("  3) %s\n", i18n.T("status.sent"))
		fmt.Printf("  4) %s\n", i18n.T("status.paid"))
		fmt.Printf("  5) %s\n", i18n.T("status.overdue"))
		fmt.Printf("  6) %s\n", i18n.T("status.cancelled"))
		fmt.Println()
		ch := c.prompt(i18n.T("prompt.choice"))
		switch ch {
		case "1":
			*status = model.StatusDraft
		case "2":
			*status = model.StatusCreated
		case "3":
			*status = model.StatusSent
		case "4":
			*status = model.StatusPaid
		case "5":
			*status = model.StatusOverdue
		case "6":
			*status = model.StatusCancelled
		default:
			return
		}
		c.updateFilterLabel(status, customerID, from, to, label)

	case "c":
		customer, goBack := c.selectCustomerWithBack()
		if goBack || customer == nil {
			return
		}
		*customerID = customer.ID
		c.updateFilterLabel(status, customerID, from, to, label)

	case "d":
		fromStr := c.promptDefault(i18n.T("prompt.date_from"), "")
		if fromStr != "" {
			if t, err := time.Parse("02.01.2006", fromStr); err == nil {
				*from = t
			}
		}
		toStr := c.promptDefault(i18n.T("prompt.date_to"), "")
		if toStr != "" {
			if t, err := time.Parse("02.01.2006", toStr); err == nil {
				*to = t
			}
		}
		c.updateFilterLabel(status, customerID, from, to, label)

	case "r":
		*status = ""
		*customerID = ""
		*from = time.Time{}
		*to = time.Time{}
		*label = ""
	}
}

func (c *CLI) updateFilterLabel(status *model.InvoiceStatus, customerID *string, from, to *time.Time, label *string) {
	var parts []string
	if *status != "" {
		parts = append(parts, c.statusName(*status))
	}
	if *customerID != "" {
		customer, _ := c.customers.GetByID(*customerID)
		if customer != nil {
			parts = append(parts, customer.Name)
		}
	}
	if !from.IsZero() || !to.IsZero() {
		dateRange := ""
		if !from.IsZero() && !to.IsZero() {
			dateRange = from.Format("02.01.2006") + " - " + to.Format("02.01.2006")
		} else if !from.IsZero() {
			dateRange = from.Format("02.01.2006") + " →"
		} else {
			dateRange = "→ " + to.Format("02.01.2006")
		}
		parts = append(parts, dateRange)
	}
	if len(parts) > 0 {
		*label = strings.Join(parts, ", ")
	} else {
		*label = ""
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
		if inv.Status == model.StatusDraft {
			fmt.Println("  " + i18n.T("action.edit_invoice"))
		}
		fmt.Println("  " + i18n.T("action.delete_invoice"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choice"))

		switch choice {
		case "0", "":
			return
		case "e", "E":
			if inv.Status == model.StatusDraft {
				c.editDraftInvoice(inv)
			}
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

func (c *CLI) editDraftInvoice(inv *model.Invoice) {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.Tf("heading.edit_invoice", inv.InvoiceNumber))
		fmt.Println()

		customer, _ := c.customers.GetByID(inv.CustomerID)
		custName := "?"
		if customer != nil {
			custName = customer.Name
		}

		fmt.Printf("  "+i18n.T("label.customer_short")+"\n", custName)
		fmt.Printf("  "+i18n.T("label.due_date_short")+"\n", inv.DueDate.Format("02.01.2006"))
		if inv.Notes != "" {
			c.printMultiline("  ", i18n.T("label.notes"), inv.Notes)
		}
		fmt.Println()

		fmt.Println("  " + i18n.T("action.change_customer"))
		fmt.Println("  " + i18n.T("action.change_due_date"))
		fmt.Println("  " + i18n.T("action.change_notes"))
		fmt.Println("  " + i18n.T("action.edit_items"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := strings.ToLower(c.prompt(i18n.T("prompt.choice")))

		switch choice {
		case "0", "":
			return
		case "c":
			newCust, goBack := c.selectCustomerWithBack()
			if !goBack && newCust != nil {
				inv.CustomerID = newCust.ID
				c.invoices.Update(inv)
				c.printSuccess(i18n.T("success.invoice_updated"))
			}
		case "d":
			dueDateStr := c.promptDefault(i18n.T("prompt.new_due_date"), inv.DueDate.Format("02.01.2006"))
			if t, err := time.Parse("02.01.2006", dueDateStr); err == nil {
				inv.DueDate = t
				c.invoices.Update(inv)
				c.printSuccess(i18n.T("success.invoice_updated"))
			}
		case "n":
			inv.Notes = c.promptDefault(i18n.T("prompt.invoice_notes"), inv.Notes)
			c.invoices.Update(inv)
			c.printSuccess(i18n.T("success.invoice_updated"))
		case "i":
			items, _ := c.invItems.GetByInvoice(inv.ID)
			c.editItemsList(&items)

			// Recalculate and save atomically
			if err := repository.WithTx(c.db.DB, func(tx *sql.Tx) error {
				// Delete old items
				if err := c.invItems.WithDB(tx).DeleteByInvoice(inv.ID); err != nil {
					return err
				}
				// Re-assign invoice ID and create catalog entries
				itemRepo := c.items.WithDB(tx)
				for i := range items {
					items[i].InvoiceID = inv.ID
					items[i].ID = "" // force new UUIDs
					if items[i].ItemID == "" {
						existing, _ := itemRepo.FindByDescription(items[i].Description)
						if existing != nil {
							items[i].ItemID = existing.ID
						} else {
							catalogItem := &model.Item{
								Description:    items[i].Description,
								DefaultPrice:   items[i].UnitPrice,
								DefaultUnit:    items[i].Unit,
								DefaultVATRate: items[i].VATRate,
								LastUsedPrice:  items[i].UnitPrice,
								LastCustomerID: inv.CustomerID,
								UsageCount:     0,
							}
							if err := itemRepo.Create(catalogItem); err != nil {
								return err
							}
							items[i].ItemID = catalogItem.ID
						}
					}
				}
				// Re-insert items
				if err := c.invItems.WithDB(tx).CreateBatch(items); err != nil {
					return err
				}

				// Update invoice totals inside transaction
				inv.Subtotal = 0
				inv.VATTotal = 0
				inv.Total = 0
				for _, item := range items {
					inv.Subtotal += item.Subtotal
					inv.VATTotal += item.VATAmount
					inv.Total += item.Total
				}
				if err := c.invoices.WithDB(tx).Update(inv); err != nil {
					return err
				}

				return nil
			}); err != nil {
				c.printError(err.Error())
				c.waitEnter()
				continue
			}

			c.printSuccess(i18n.T("success.invoice_updated"))
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

	var bankAcc *model.BankAccount
	if inv.BankAccountID != "" {
		bankAcc, err = c.bankAccs.GetByID(inv.BankAccountID)
		if err != nil || bankAcc == nil {
			c.printError(i18n.T("error.load_bank_account"))
			return
		}
	}
	if bankAcc == nil {
		bankAcc = &model.BankAccount{} // empty sentinel
	}

	items, err := c.invItems.GetByInvoice(inv.ID)
	if err != nil {
		c.printError(i18n.T("error.load_items"))
		return
	}

	hasBankInfo := bankAcc.AccountNumber != "" || bankAcc.IBAN != ""

	fmt.Println(i18n.T("info.generating_pdf"))

	data := &service.InvoiceData{
		Invoice:     inv,
		Supplier:    supplier,
		Customer:    customer,
		BankAccount: bankAcc,
		Items:       items,
	}

	opts := &service.TemplateOptions{
		ShowLogo:    true,
		ShowQR:      hasBankInfo,
		ShowNotes:   true,
		QRType:      bankAcc.QRType,
		HasBankInfo: hasBankInfo,
	}

	// Resolve template: use invoice's template, fall back to user's default
	templateID := inv.TemplateID
	if templateID == "" {
		if defTmpl, err := c.templates.GetDefault(); err == nil && defTmpl != nil {
			templateID = defTmpl.ID
		}
	}

	// Look up template settings and YAML source
	yamlSource := ""
	templateCode := templateID
	if tmpl, err := c.templates.GetByID(templateID); err == nil && tmpl != nil {
		opts.ShowLogo = tmpl.ShowLogo
		opts.ShowQR = tmpl.ShowQR
		opts.ShowNotes = tmpl.ShowNotes
		templateCode = tmpl.TemplateCode
		if !tmpl.IsBuiltin && tmpl.YAMLSource != "" {
			yamlSource = tmpl.YAMLSource
		}
	}

	pdfPath, err := c.pdfService.GenerateInvoiceWithYAML(data, templateCode, yamlSource, opts)
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
	fmt.Printf("  5) %s\n", i18n.T("status.cancelled"))
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
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.duplicate_invoice"))
	fmt.Println()

	// List invoices to pick from
	invoices, err := c.invoices.List("", "")
	if err != nil || len(invoices) == 0 {
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
		fmt.Printf("  %d) %s %s | %s | %10.2f %s\n",
			i+1, c.statusIcon(inv.Status), inv.InvoiceNumber, custName, inv.Total, inv.Currency)
	}
	fmt.Println()
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.select_invoice"))
	if choice == "0" || choice == "" {
		return
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > len(invoices) {
		return
	}

	sourceInv := invoices[idx-1]
	sourceItems, _ := c.invItems.GetByInvoice(sourceInv.ID)

	// Load related data
	customer, _ := c.customers.GetByID(sourceInv.CustomerID)
	if customer == nil {
		c.printError(i18n.T("error.load_customer"))
		c.waitEnter()
		return
	}

	supplier, _ := c.suppliers.GetByID(sourceInv.SupplierID)
	if supplier == nil {
		c.printError(i18n.T("error.load_supplier"))
		c.waitEnter()
		return
	}

	// Show source info
	fmt.Println()
	itemDescs := make([]string, 0, len(sourceItems))
	for _, it := range sourceItems {
		itemDescs = append(itemDescs, fmt.Sprintf("%s (%.2f)", it.Description, it.UnitPrice))
	}
	fmt.Println(i18n.Tf("label.source_invoice", sourceInv.InvoiceNumber, customer.Name, sourceInv.Total, sourceInv.Currency))
	fmt.Println(i18n.Tf("label.source_items", len(sourceItems), strings.Join(itemDescs, ", ")))
	fmt.Println()

	fmt.Println("  " + i18n.T("action.quick_duplicate"))
	fmt.Println("  " + i18n.T("action.edit_before_save"))
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	mode := strings.ToLower(c.prompt(i18n.T("prompt.choice")))

	if mode == "0" || mode == "" {
		return
	}

	// Generate new invoice number (user can override)
	invNumber, err := c.invoices.GetNextNumber(supplier.ID, supplier.InvoicePrefix)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}
	invNumber = c.promptDefault(i18n.T("prompt.invoice_number"), invNumber)

	// Create new invoice copying fields from source
	newInv := model.NewInvoice(sourceInv.SupplierID, sourceInv.CustomerID, sourceInv.BankAccountID)
	newInv.InvoiceNumber = invNumber
	newInv.VariableSymbol = repository.GenerateVariableSymbol(invNumber)
	newInv.Currency = sourceInv.Currency
	newInv.PaymentMethod = sourceInv.PaymentMethod
	newInv.ExchangeRate = sourceInv.ExchangeRate
	newInv.Language = sourceInv.Language
	dupDueDays := customer.DefaultDueDays
	if dupDueDays == 0 {
		dupDueDays = c.getDefaultDueDaysInt()
	}
	newInv.DueDate = time.Now().AddDate(0, 0, dupDueDays)
	newInv.Notes = sourceInv.Notes

	// Deep-copy items (clear IDs for new UUIDs)
	newItems := make([]model.InvoiceItem, len(sourceItems))
	for i, src := range sourceItems {
		newItems[i] = src
		newItems[i].ID = ""
		newItems[i].InvoiceID = ""
	}

	if mode == "q" {
		// Quick duplicate — save immediately
		if len(newItems) == 0 {
			c.printError(i18n.T("error.invoice_no_items"))
			c.waitEnter()
			return
		}
		c.saveInvoiceWithSummary(newInv, newItems, customer)
		return
	}

	if mode == "e" {
		// Edit mode — review each field
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.duplicate_invoice"))
		fmt.Println()

		// Edit invoice details with defaults from source
		newCust, goBack := c.selectCustomerWithBack()
		if goBack {
			return
		}
		if newCust != nil {
			newInv.CustomerID = newCust.ID
			customer = newCust
			editDueDays := customer.DefaultDueDays
			if editDueDays == 0 {
				editDueDays = c.getDefaultDueDaysInt()
			}
			newInv.DueDate = time.Now().AddDate(0, 0, editDueDays)
		}

		dueDateStr := c.promptDefault(i18n.T("prompt.new_due_date"), newInv.DueDate.Format("02.01.2006"))
		if t, err := time.Parse("02.01.2006", dueDateStr); err == nil {
			newInv.DueDate = t
		}

		newInv.Notes = c.promptDefault(i18n.T("prompt.invoice_notes"), newInv.Notes)

		// Edit items
		c.editItemsList(&newItems)

		if len(newItems) == 0 {
			c.printError(i18n.T("error.invoice_no_items"))
			c.waitEnter()
			return
		}

		c.saveInvoiceWithSummary(newInv, newItems, customer)
	}
}

func (c *CLI) editItemsList(items *[]model.InvoiceItem) {
	for {
		c.clearScreen()
		fmt.Printf("--- %s (%d) ---\n", i18n.T("label.items"), len(*items))
		fmt.Println()

		if len(*items) == 0 {
			fmt.Println("  " + i18n.T("info.no_items"))
		} else {
			for i, item := range *items {
				fmt.Printf("  e%d) %.0fx %s @ %.2f = %.2f    x%d) %s\n",
					i+1, item.Quantity, item.Description, item.UnitPrice, item.Total,
					i+1, i18n.T("label.remove"))
			}
		}

		fmt.Println()
		fmt.Println("  " + i18n.T("action.remove_all_items"))
		fmt.Println("  " + i18n.T("action.add_item_edit"))
		fmt.Println("  " + i18n.T("action.continue"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := strings.ToLower(c.prompt(i18n.T("prompt.choice")))

		if choice == "c" {
			return
		}
		if choice == "0" {
			return
		}

		if choice == "r" {
			if c.confirm(i18n.T("confirm.remove_all_items")) {
				*items = nil
			}
			continue
		}

		if choice == "n" || choice == "a" {
			item, _ := c.addInvoiceItemWithBack("", len(*items), nil)
			if item != nil {
				*items = append(*items, *item)
			}
			continue
		}

		// Handle e/eN (edit) and x/xN (remove)
		if strings.HasPrefix(choice, "e") {
			numStr := choice[1:]
			if numStr == "" {
				numStr = c.prompt(i18n.T("prompt.which_item"))
			}
			idx := 0
			fmt.Sscanf(numStr, "%d", &idx)
			if idx > 0 && idx <= len(*items) {
				c.editSingleItem(&(*items)[idx-1])
			}
			continue
		}

		if strings.HasPrefix(choice, "x") {
			numStr := choice[1:]
			if numStr == "" {
				numStr = c.prompt(i18n.T("prompt.which_item"))
			}
			idx := 0
			fmt.Sscanf(numStr, "%d", &idx)
			if idx > 0 && idx <= len(*items) {
				if c.confirm(i18n.T("confirm.remove_item")) {
					*items = append((*items)[:idx-1], (*items)[idx:]...)
				}
			}
			continue
		}
	}
}

func (c *CLI) editSingleItem(item *model.InvoiceItem) {
	fmt.Println()
	item.Description = c.promptDefaultMaxLen(i18n.T("prompt.item_description"), item.Description, model.MaxDescriptionLen)
	item.Quantity = c.promptFloat(i18n.T("prompt.quantity"), item.Quantity)
	item.Unit = c.promptDefault(i18n.T("prompt.unit"), item.Unit)
	item.UnitPrice = c.promptFloat(i18n.T("prompt.unit_price"), item.UnitPrice)
	item.VATRate = c.promptFloat(i18n.T("prompt.vat_rate"), item.VATRate)
	item.Calculate()

	fmt.Printf("  → %.0f %s × %.2f = %.2f\n",
		item.Quantity, item.Unit, item.UnitPrice, item.Total)
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

	type sectionInfo struct {
		label string
		start int
	}

	var entries []catalogEntry
	var sections []sectionInfo
	seen := make(map[string]bool)

	// Customer items section
	if len(customerItems) > 0 {
		sections = append(sections, sectionInfo{i18n.T("label.customer_items"), len(entries)})
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

	// Recently used items section
	recentItems, _ := c.items.GetRecentlyUsed(5)
	var recentEntries []catalogEntry
	for _, item := range recentItems {
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		recentEntries = append(recentEntries, catalogEntry{item: item})
	}
	if len(recentEntries) > 0 {
		sections = append(sections, sectionInfo{i18n.T("label.recent_items"), len(entries)})
		entries = append(entries, recentEntries...)
	}

	// Global most-used items section
	var globalEntries []catalogEntry
	for _, item := range globalItems {
		if seen[item.ID] {
			continue
		}
		globalEntries = append(globalEntries, catalogEntry{item: item})
	}
	if len(globalEntries) > 0 {
		sections = append(sections, sectionInfo{i18n.T("label.global_items"), len(entries)})
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

	sectionIdx := 0
	for i, entry := range entries {
		// Check if we need to print a section header
		if sectionIdx < len(sections) && sections[sectionIdx].start == i {
			if i > 0 {
				fmt.Println()
			}
			fmt.Println("  " + sections[sectionIdx].label)
			sectionIdx++
		}

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
