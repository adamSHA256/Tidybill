package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) suppliersMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.suppliers"))
		fmt.Println()

		suppliers, err := c.suppliers.List()
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		for i, s := range suppliers {
			def := ""
			if s.IsDefault {
				def = " " + i18n.T("label.default_upper")
			}
			fmt.Printf("  %d) %s%s\n", i+1, s.Name, def)
		}

		fmt.Println()
		fmt.Println("  " + i18n.T("action.new_supplier"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch choice {
		case "0", "":
			return
		case "n", "N":
			c.createSupplier()
		default:
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			if idx > 0 && idx <= len(suppliers) {
				c.editSupplier(suppliers[idx-1])
			}
		}
	}
}

func (c *CLI) createSupplier() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.new_supplier"))
	fmt.Println()

	supplier := model.NewSupplier()
	supplier.IsDefault = false

	supplier.Name = c.prompt(i18n.T("prompt.company_name"))
	if supplier.Name == "" {
		c.printError(i18n.T("error.name_required"))
		c.waitEnter()
		return
	}

	supplier.Street = c.prompt(i18n.T("prompt.street"))
	supplier.City = c.prompt(i18n.T("prompt.city"))
	supplier.ZIP = c.prompt(i18n.T("prompt.zip"))
	supplier.Country = c.promptDefault(i18n.T("prompt.country"), "CZ")
	supplier.ICO = c.prompt(i18n.T("prompt.ico"))
	supplier.DIC = c.prompt(i18n.T("prompt.dic"))
	supplier.Phone = c.prompt(i18n.T("prompt.phone"))
	supplier.Email = c.prompt(i18n.T("prompt.email"))
	supplier.Website = c.prompt(i18n.T("prompt.website"))
	supplier.InvoicePrefix = c.promptDefault(i18n.T("prompt.invoice_prefix_short"), "VF")

	if err := c.suppliers.Create(supplier); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Ask for bank account
	fmt.Println()
	if c.confirm(i18n.T("confirm.add_bank_account")) {
		c.addBankAccount(supplier.ID)
	}

	c.printSuccess(i18n.T("success.supplier_created"))
	c.waitEnter()
}

func (c *CLI) editSupplier(s *model.Supplier) {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.Tf("heading.supplier_detail", s.Name))
		fmt.Println()
		fmt.Printf("  "+i18n.T("label.address")+"\n", s.Street, s.ZIP, s.City, s.Country)
		fmt.Printf("  "+i18n.T("label.ico")+"\n", s.ICO)
		fmt.Printf("  "+i18n.T("label.dic")+"\n", s.DIC)
		fmt.Printf("  "+i18n.T("label.phone")+"\n", s.Phone)
		fmt.Printf("  "+i18n.T("label.email")+"\n", s.Email)
		fmt.Printf("  "+i18n.T("label.website")+"\n", s.Website)
		fmt.Printf("  "+i18n.T("label.prefix")+"\n", s.InvoicePrefix)
		fmt.Printf("  "+i18n.T("label.is_default")+"\n", s.IsDefault)
		fmt.Println()

		// Show bank accounts
		accounts, _ := c.bankAccs.GetBySupplier(s.ID)
		if len(accounts) > 0 {
			fmt.Println("  " + i18n.T("label.bank_accounts"))
			for _, acc := range accounts {
				def := ""
				if acc.IsDefault {
					def = " " + i18n.T("label.default_lower")
				}
				fmt.Printf("    - %s: %s (%s)%s\n", acc.Name, acc.AccountNumber, acc.Currency, def)
			}
			fmt.Println()
		}

		fmt.Println("  " + i18n.T("action.edit_details"))
		fmt.Println("  " + i18n.T("action.add_bank_account"))
		if !s.IsDefault {
			fmt.Println("  " + i18n.T("action.set_default"))
		}
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch choice {
		case "0", "":
			return
		case "e", "E":
			c.editSupplierDetails(s)
		case "b", "B":
			c.addBankAccount(s.ID)
		case "d", "D":
			if !s.IsDefault {
				c.suppliers.SetDefault(s.ID)
				s.IsDefault = true
				c.currentSupp = s.ID
				c.printSuccess(i18n.T("success.set_as_default"))
			}
		}
	}
}

func (c *CLI) editSupplierDetails(s *model.Supplier) {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.edit_supplier"))
	fmt.Println(i18n.T("prompt.leave_empty_hint"))
	fmt.Println()

	s.Name = c.promptDefault(i18n.T("prompt.name"), s.Name)
	s.Street = c.promptDefault(i18n.T("prompt.street_short"), s.Street)
	s.City = c.promptDefault(i18n.T("prompt.city"), s.City)
	s.ZIP = c.promptDefault(i18n.T("prompt.zip"), s.ZIP)
	s.Country = c.promptDefault(i18n.T("prompt.country"), s.Country)
	s.ICO = c.promptDefault(i18n.T("prompt.ico"), s.ICO)
	s.DIC = c.promptDefault(i18n.T("prompt.dic"), s.DIC)
	s.Phone = c.promptDefault(i18n.T("prompt.phone"), s.Phone)
	s.Email = c.promptDefault(i18n.T("prompt.email"), s.Email)
	s.Website = c.promptDefault(i18n.T("prompt.website"), s.Website)
	s.InvoicePrefix = c.promptDefault(i18n.T("prompt.invoice_prefix_short"), s.InvoicePrefix)

	if err := c.suppliers.Update(s); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.supplier_updated"))
	}
	c.waitEnter()
}

func (c *CLI) addBankAccount(supplierID string) {
	fmt.Println()
	fmt.Printf("=== %s ===\n", i18n.T("heading.new_bank_account"))
	fmt.Println()

	acc := model.NewBankAccount(supplierID)

	acc.Name = c.promptDefault(i18n.T("prompt.account_name"), i18n.T("default.main_account"))
	acc.AccountNumber = c.prompt(i18n.T("prompt.account_number"))
	acc.IBAN = c.prompt(i18n.T("prompt.iban"))
	acc.SWIFT = c.prompt(i18n.T("prompt.swift"))
	acc.Currency = c.promptDefault(i18n.T("prompt.currency"), "CZK")

	// Check if this is the first account for supplier
	existing, _ := c.bankAccs.GetBySupplier(supplierID)
	if len(existing) == 0 {
		acc.IsDefault = true
	} else {
		acc.IsDefault = c.confirm(i18n.T("confirm.set_as_default"))
	}

	if err := c.bankAccs.Create(acc); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.account_added"))
	}
}
