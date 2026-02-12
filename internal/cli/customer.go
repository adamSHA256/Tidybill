package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) customersMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.customers"))
		fmt.Println()

		customers, err := c.customers.List()
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		if len(customers) == 0 {
			fmt.Println("  " + i18n.T("info.no_customers"))
		} else {
			for i, cust := range customers {
				fmt.Printf("  %d) %s", i+1, cust.Name)
				if cust.ICO != "" {
					fmt.Printf(" (%s: %s)", i18n.T("prompt.ico"), cust.ICO)
				}
				fmt.Println()
			}
		}

		fmt.Println()
		fmt.Println("  " + i18n.T("action.new_customer"))
		fmt.Println("  " + i18n.T("action.search"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch choice {
		case "0", "":
			return
		case "n", "N":
			c.createCustomer()
		case "h", "H":
			c.searchCustomers()
		default:
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			if idx > 0 && idx <= len(customers) {
				c.editCustomer(customers[idx-1])
			}
		}
	}
}

func (c *CLI) createCustomer() *model.Customer {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.new_customer"))
	fmt.Println(i18n.T("prompt.enter_0_back"))
	fmt.Println()

	cust := model.NewCustomer()

	name, goBack := c.promptWithBack(i18n.T("prompt.company_name"))
	if goBack {
		return nil
	}
	if name == "" {
		c.printError(i18n.T("error.name_required"))
		c.waitEnter()
		return nil
	}
	cust.Name = name

	cust.Street = c.prompt(i18n.T("prompt.street"))
	cust.City = c.prompt(i18n.T("prompt.city"))
	cust.ZIP = c.prompt(i18n.T("prompt.zip"))
	cust.Region = c.prompt(i18n.T("prompt.region"))
	cust.Country = c.promptDefault(i18n.T("prompt.country"), "CZ")
	cust.ICO = c.prompt(i18n.T("prompt.ico"))
	cust.DIC = c.prompt(i18n.T("prompt.dic"))
	cust.Email = c.prompt(i18n.T("prompt.email"))
	cust.Phone = c.prompt(i18n.T("prompt.phone"))
	cust.DefaultDueDays = c.promptInt(i18n.T("prompt.default_due_days"), 14)
	cust.Notes = c.prompt(i18n.T("prompt.notes"))

	if err := c.customers.Create(cust); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return nil
	}

	c.printSuccess(i18n.T("success.customer_created"))
	c.waitEnter()
	return cust
}

func (c *CLI) editCustomer(cust *model.Customer) {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.Tf("heading.customer_detail", cust.Name))
		fmt.Println()
		fmt.Printf("  "+i18n.T("label.address")+"\n", cust.Street, cust.ZIP, cust.City, cust.Country)
		if cust.Region != "" {
			fmt.Printf("  "+i18n.T("label.region")+"\n", cust.Region)
		}
		fmt.Printf("  "+i18n.T("label.ico")+"\n", cust.ICO)
		fmt.Printf("  "+i18n.T("label.dic")+"\n", cust.DIC)
		fmt.Printf("  "+i18n.T("label.email_full")+"\n", cust.Email)
		fmt.Printf("  "+i18n.T("label.phone_full")+"\n", cust.Phone)
		fmt.Printf("  "+i18n.T("label.due_days")+"\n", cust.DefaultDueDays)
		if cust.Notes != "" {
			c.printMultiline("  ", i18n.T("label.notes"), cust.Notes)
		}
		fmt.Println()

		fmt.Println("  " + i18n.T("action.edit_details"))
		fmt.Println("  " + i18n.T("action.notes"))
		fmt.Println("  " + i18n.T("action.show_invoices"))
		fmt.Println("  " + i18n.T("action.delete_customer"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch choice {
		case "0", "":
			return
		case "e", "E":
			c.editCustomerDetails(cust)
		case "n", "N":
			c.editCustomerNotes(cust)
		case "f", "F":
			c.listCustomerInvoices(cust)
		case "x", "X":
			if c.confirm(i18n.T("confirm.delete_customer")) {
				if err := c.customers.Delete(cust.ID); err != nil {
					c.printError(err.Error())
					c.waitEnter()
				} else {
					c.printSuccess(i18n.T("success.customer_deleted"))
					c.waitEnter()
					return
				}
			}
		}
	}
}

func (c *CLI) editCustomerDetails(cust *model.Customer) {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.edit_customer"))
	fmt.Println(i18n.T("prompt.leave_empty_hint"))
	fmt.Println()

	cust.Name = c.promptDefault(i18n.T("prompt.name"), cust.Name)
	cust.Street = c.promptDefault(i18n.T("prompt.street_short"), cust.Street)
	cust.City = c.promptDefault(i18n.T("prompt.city"), cust.City)
	cust.ZIP = c.promptDefault(i18n.T("prompt.zip"), cust.ZIP)
	cust.Region = c.promptDefault(i18n.T("prompt.region"), cust.Region)
	cust.Country = c.promptDefault(i18n.T("prompt.country"), cust.Country)
	cust.ICO = c.promptDefault(i18n.T("prompt.ico"), cust.ICO)
	cust.DIC = c.promptDefault(i18n.T("prompt.dic"), cust.DIC)
	cust.Email = c.promptDefault(i18n.T("prompt.email"), cust.Email)
	cust.Phone = c.promptDefault(i18n.T("prompt.phone"), cust.Phone)
	cust.DefaultDueDays = c.promptInt(i18n.T("prompt.due_days"), cust.DefaultDueDays)
	cust.Notes = c.promptDefault(i18n.T("prompt.notes"), cust.Notes)

	if err := c.customers.Update(cust); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.customer_updated"))
	}
	c.waitEnter()
}

func (c *CLI) searchCustomers() {
	query := c.prompt(i18n.T("prompt.search_name_ico"))
	if query == "" {
		return
	}

	customers, err := c.customers.Search(query)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(customers) == 0 {
		fmt.Println(i18n.T("info.nothing_found"))
		c.waitEnter()
		return
	}

	fmt.Println()
	for i, cust := range customers {
		fmt.Printf("  %d) %s", i+1, cust.Name)
		if cust.ICO != "" {
			fmt.Printf(" (%s: %s)", i18n.T("prompt.ico"), cust.ICO)
		}
		fmt.Println()
	}

	fmt.Println()
	choice := c.prompt(i18n.T("prompt.select_for_detail"))
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(customers) {
		c.editCustomer(customers[idx-1])
	}
}

func (c *CLI) listCustomerInvoices(cust *model.Customer) {
	invoices, err := c.invoices.List("", cust.ID)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.Tf("heading.customer_invoices", cust.Name))
	fmt.Println()

	if len(invoices) == 0 {
		fmt.Println(i18n.T("info.no_invoices"))
	} else {
		for _, inv := range invoices {
			fmt.Printf("  %s | %s | %.2f %s | %s\n",
				inv.InvoiceNumber,
				inv.IssueDate.Format("02.01.2006"),
				inv.Total,
				inv.Currency,
				inv.Status)
		}
	}

	c.waitEnter()
}

func (c *CLI) selectCustomer() *model.Customer {
	cust, _ := c.selectCustomerWithBack()
	return cust
}

func (c *CLI) selectCustomerWithBack() (*model.Customer, bool) {
	customers, _ := c.customers.List()

	if len(customers) == 0 {
		fmt.Println(i18n.T("info.no_customers_create"))
		cust := c.createCustomer()
		return cust, cust == nil
	}

	fmt.Println()
	fmt.Println(i18n.T("prompt.select_customer"))
	for i, cust := range customers {
		fmt.Printf("  %d) %s\n", i+1, cust.Name)
	}
	fmt.Println("  " + i18n.T("action.new_customer"))
	fmt.Println("  " + i18n.T("action.back"))
	fmt.Println()

	choice := c.prompt(i18n.T("prompt.choice"))

	if choice == "0" {
		return nil, true
	}

	if choice == "n" || choice == "N" {
		cust := c.createCustomer()
		return cust, cust == nil
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(customers) {
		return customers[idx-1], false
	}

	return nil, false
}

func (c *CLI) editCustomerNotes(cust *model.Customer) {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.Tf("heading.customer_detail", cust.Name))
	fmt.Printf("\n  %s\n\n", i18n.T("prompt.notes"))

	cust.Notes = c.editNotes(cust.Notes)

	if err := c.customers.Update(cust); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.notes_saved"))
	}
	c.waitEnter()
}
