package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) suppliersMenu() {
	for {
		c.clearScreen()
		fmt.Println("=== DODAVATELÉ (vaše firmy) ===")
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
				def = " [VÝCHOZÍ]"
			}
			fmt.Printf("  %d) %s%s\n", i+1, s.Name, def)
		}

		fmt.Println()
		fmt.Println("  N) Nový dodavatel")
		fmt.Println("  0) Zpět")
		fmt.Println()

		choice := c.prompt("Vyberte možnost")

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
	fmt.Println("=== NOVÝ DODAVATEL ===")
	fmt.Println()

	supplier := model.NewSupplier()
	supplier.IsDefault = false

	supplier.Name = c.prompt("Název firmy / Jméno")
	if supplier.Name == "" {
		c.printError("Název je povinný")
		c.waitEnter()
		return
	}

	supplier.Street = c.prompt("Ulice a číslo")
	supplier.City = c.prompt("Město")
	supplier.ZIP = c.prompt("PSČ")
	supplier.Country = c.promptDefault("Země", "CZ")
	supplier.ICO = c.prompt("IČO")
	supplier.DIC = c.prompt("DIČ")
	supplier.Phone = c.prompt("Telefon")
	supplier.Email = c.prompt("E-mail")
	supplier.Website = c.prompt("Website")
	supplier.InvoicePrefix = c.promptDefault("Prefix faktur", "VF")

	if err := c.suppliers.Create(supplier); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Ask for bank account
	fmt.Println()
	if c.confirm("Přidat bankovní účet?") {
		c.addBankAccount(supplier.ID)
	}

	c.printSuccess("Dodavatel byl vytvořen")
	c.waitEnter()
}

func (c *CLI) editSupplier(s *model.Supplier) {
	for {
		c.clearScreen()
		fmt.Printf("=== DODAVATEL: %s ===\n", s.Name)
		fmt.Println()
		fmt.Printf("  Adresa:  %s, %s %s, %s\n", s.Street, s.ZIP, s.City, s.Country)
		fmt.Printf("  IČO:     %s\n", s.ICO)
		fmt.Printf("  DIČ:     %s\n", s.DIC)
		fmt.Printf("  Tel:     %s\n", s.Phone)
		fmt.Printf("  E-mail:  %s\n", s.Email)
		fmt.Printf("  Website: %s\n", s.Website)
		fmt.Printf("  Prefix:  %s\n", s.InvoicePrefix)
		fmt.Printf("  Výchozí: %v\n", s.IsDefault)
		fmt.Println()

		// Show bank accounts
		accounts, _ := c.bankAccs.GetBySupplier(s.ID)
		if len(accounts) > 0 {
			fmt.Println("  Bankovní účty:")
			for _, acc := range accounts {
				def := ""
				if acc.IsDefault {
					def = " [výchozí]"
				}
				fmt.Printf("    - %s: %s (%s)%s\n", acc.Name, acc.AccountNumber, acc.Currency, def)
			}
			fmt.Println()
		}

		fmt.Println("  E) Upravit údaje")
		fmt.Println("  B) Přidat bankovní účet")
		if !s.IsDefault {
			fmt.Println("  D) Nastavit jako výchozí")
		}
		fmt.Println("  0) Zpět")
		fmt.Println()

		choice := c.prompt("Vyberte možnost")

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
				c.printSuccess("Nastaveno jako výchozí")
			}
		}
	}
}

func (c *CLI) editSupplierDetails(s *model.Supplier) {
	c.clearScreen()
	fmt.Println("=== ÚPRAVA DODAVATELE ===")
	fmt.Println("(Nechte prázdné pro zachování stávající hodnoty)")
	fmt.Println()

	s.Name = c.promptDefault("Název", s.Name)
	s.Street = c.promptDefault("Ulice", s.Street)
	s.City = c.promptDefault("Město", s.City)
	s.ZIP = c.promptDefault("PSČ", s.ZIP)
	s.Country = c.promptDefault("Země", s.Country)
	s.ICO = c.promptDefault("IČO", s.ICO)
	s.DIC = c.promptDefault("DIČ", s.DIC)
	s.Phone = c.promptDefault("Telefon", s.Phone)
	s.Email = c.promptDefault("E-mail", s.Email)
	s.Website = c.promptDefault("Website", s.Website)
	s.InvoicePrefix = c.promptDefault("Prefix faktur", s.InvoicePrefix)

	if err := c.suppliers.Update(s); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess("Dodavatel byl aktualizován")
	}
	c.waitEnter()
}

func (c *CLI) addBankAccount(supplierID string) {
	fmt.Println()
	fmt.Println("=== NOVÝ BANKOVNÍ ÚČET ===")
	fmt.Println()

	acc := model.NewBankAccount(supplierID)

	acc.Name = c.promptDefault("Název účtu", "Hlavní účet")
	acc.AccountNumber = c.prompt("Číslo účtu (např. 1234567890/0100)")
	acc.IBAN = c.prompt("IBAN")
	acc.SWIFT = c.prompt("SWIFT/BIC")
	acc.Currency = c.promptDefault("Měna", "CZK")

	// Check if this is the first account for supplier
	existing, _ := c.bankAccs.GetBySupplier(supplierID)
	if len(existing) == 0 {
		acc.IsDefault = true
	} else {
		acc.IsDefault = c.confirm("Nastavit jako výchozí?")
	}

	if err := c.bankAccs.Create(acc); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess("Účet byl přidán")
	}
}
