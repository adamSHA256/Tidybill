package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) firstRunWizard() error {
	// Language selection before wizard — labels in all languages so user can read
	c.clearScreen()
	fmt.Println("  Choose language / Vyberte jazyk / Vyberte jazyk:")
	fmt.Println()
	langs := i18n.AvailableLanguages()
	currentLang := i18n.GetLang()
	for idx, lang := range langs {
		marker := "  "
		if lang == currentLang {
			marker = "* "
		}
		fmt.Printf("  %s%d) %s\n", marker, idx+1, langName(lang))
	}
	fmt.Println()
	langChoice := c.prompt("->")
	switch langChoice {
	case "1":
		i18n.SetLang(i18n.CS)
	case "2":
		i18n.SetLang(i18n.SK)
	case "3":
		i18n.SetLang(i18n.EN)
	}
	c.settings.Set("language", string(i18n.GetLang()))

	c.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Printf("║               %-45s║\n", i18n.T("wizard.welcome_title"))
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                            ║")
	fmt.Printf("║  %-58s║\n", i18n.T("wizard.no_data"))
	fmt.Printf("║  %-58s║\n", i18n.T("wizard.setup_prompt"))
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("=== %s ===\n", i18n.T("wizard.supplier_details"))
	fmt.Println()

	supplier := model.NewSupplier()

	supplier.Name = c.prompt(i18n.T("prompt.company_name"))
	if supplier.Name == "" {
		return fmt.Errorf(i18n.T("error.name_required_lower"))
	}

	supplier.Street = c.prompt(i18n.T("prompt.street"))
	supplier.City = c.prompt(i18n.T("prompt.city"))
	supplier.ZIP = c.prompt(i18n.T("prompt.zip"))
	supplier.Country = c.promptDefault(i18n.T("prompt.country"), "CZ")
	supplier.ICO = c.prompt(i18n.T("prompt.ico"))
	supplier.DIC = c.prompt(i18n.T("prompt.dic_with_hint"))
	supplier.Phone = c.prompt(i18n.T("prompt.phone"))
	supplier.Email = c.prompt(i18n.T("prompt.email"))
	supplier.Website = c.prompt(i18n.T("prompt.website"))

	if supplier.DIC != "" {
		supplier.IsVATPayer = c.confirm(i18n.T("confirm.vat_payer"))
	}

	supplier.InvoicePrefix = c.promptDefault(i18n.T("prompt.invoice_prefix"), "VF")

	fmt.Println()
	fmt.Printf("=== %s ===\n", i18n.T("wizard.bank_account"))
	fmt.Println()

	bankAcc := model.NewBankAccount("")
	bankAcc.Name = c.promptDefault(i18n.T("prompt.account_name"), i18n.T("default.main_account"))
	bankAcc.AccountNumber = c.prompt(i18n.T("prompt.account_number"))
	bankAcc.IBAN = c.prompt(i18n.T("prompt.iban"))
	bankAcc.Currency = c.promptDefault(i18n.T("prompt.currency"), "CZK")

	fmt.Println()
	fmt.Printf("=== %s ===\n", i18n.T("wizard.pdf_directory"))
	fmt.Println()
	fmt.Println(i18n.T("wizard.pdf_dir_prompt"))
	pdfDir := c.promptDefault(i18n.T("prompt.dir_pdfs"), c.cfg.PDFDir)

	fmt.Println()
	fmt.Printf("=== %s ===\n", i18n.T("wizard.summary"))
	fmt.Println()
	fmt.Println(i18n.Tf("label.company", supplier.Name))
	fmt.Println(i18n.Tf("label.address_short", supplier.Street, supplier.ZIP, supplier.City))
	fmt.Println(i18n.Tf("label.ico", supplier.ICO))
	fmt.Println(i18n.Tf("label.account", bankAcc.AccountNumber))
	fmt.Println(i18n.Tf("label.pdf_dir", pdfDir))
	fmt.Println()

	if !c.confirm(i18n.T("confirm.save_data")) {
		return fmt.Errorf(i18n.T("error.cancelled_by_user"))
	}

	// Save supplier
	if err := c.suppliers.Create(supplier); err != nil {
		return fmt.Errorf(i18n.T("error.save_failed"), err)
	}

	// Save bank account
	bankAcc.SupplierID = supplier.ID
	if err := c.bankAccs.Create(bankAcc); err != nil {
		return fmt.Errorf(i18n.T("error.save_account_failed"), err)
	}

	// Save PDF directory if changed from default
	if pdfDir != c.cfg.PDFDir {
		c.settings.Set("dir.pdfs", pdfDir)
		c.cfg.PDFDir = pdfDir
	}

	c.currentSupp = supplier.ID

	c.printSuccess(i18n.T("success.profile_created"))
	c.waitEnter()

	return nil
}
