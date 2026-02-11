package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) firstRunWizard() error {
	c.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║               VÍTEJTE V TIDYBILL                           ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                            ║")
	fmt.Println("║  Nebyla nalezena žádná data.                               ║")
	fmt.Println("║  Pojďme nastavit váš první firemní profil.                 ║")
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("=== Údaje dodavatele (vaše firma) ===")
	fmt.Println()

	supplier := model.NewSupplier()

	supplier.Name = c.prompt("Název firmy / Jméno")
	if supplier.Name == "" {
		return fmt.Errorf("název je povinný")
	}

	supplier.Street = c.prompt("Ulice a číslo")
	supplier.City = c.prompt("Město")
	supplier.ZIP = c.prompt("PSČ")
	supplier.Country = c.promptDefault("Země", "CZ")
	supplier.ICO = c.prompt("IČO")
	supplier.DIC = c.prompt("DIČ (pokud jste plátce DPH, jinak nechte prázdné)")
	supplier.Phone = c.prompt("Telefon")
	supplier.Email = c.prompt("E-mail")
	supplier.Website= c.prompt("Website")

	if supplier.DIC != "" {
		supplier.IsVATPayer = c.confirm("Jste plátce DPH?")
	}

	supplier.InvoicePrefix = c.promptDefault("Prefix čísel faktur", "VF")

	fmt.Println()
	fmt.Println("=== Bankovní účet ===")
	fmt.Println()

	bankAcc := model.NewBankAccount("")
	bankAcc.Name = c.promptDefault("Název účtu", "Hlavní účet")
	bankAcc.AccountNumber = c.prompt("Číslo účtu (např. 1234567890/0100)")
	bankAcc.IBAN = c.prompt("IBAN")
	bankAcc.Currency = c.promptDefault("Měna", "CZK")

	fmt.Println()
	fmt.Println("=== Souhrn ===")
	fmt.Println()
	fmt.Printf("Firma:    %s\n", supplier.Name)
	fmt.Printf("Adresa:   %s, %s %s\n", supplier.Street, supplier.ZIP, supplier.City)
	fmt.Printf("IČO:      %s\n", supplier.ICO)
	fmt.Printf("Účet:     %s\n", bankAcc.AccountNumber)
	fmt.Println()

	if !c.confirm("Uložit tyto údaje?") {
		return fmt.Errorf("zrušeno uživatelem")
	}

	// Save supplier
	if err := c.suppliers.Create(supplier); err != nil {
		return fmt.Errorf("chyba při ukládání: %w", err)
	}

	// Save bank account
	bankAcc.SupplierID = supplier.ID
	if err := c.bankAccs.Create(bankAcc); err != nil {
		return fmt.Errorf("chyba při ukládání účtu: %w", err)
	}

	c.currentSupp = supplier.ID

	c.printSuccess("Profil byl úspěšně vytvořen!")
	c.waitEnter()

	return nil
}
