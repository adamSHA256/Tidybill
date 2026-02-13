package cli

import (
	"fmt"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
	"github.com/adamSHA256/tidybill/internal/service"
)

func (c *CLI) templatesMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n\n", i18n.T("heading.pdf_templates"))

		templates, err := c.templates.List()
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		fmt.Printf("  %s:\n", i18n.T("templates.available"))
		for i, t := range templates {
			badge := ""
			if t.IsDefault {
				badge = fmt.Sprintf(" [%s]", i18n.T("templates.active"))
			}
			fmt.Printf("  %d) %-15s%s  - %s\n", i+1, t.Name, badge, t.Description)
		}
		fmt.Println()

		fmt.Printf("  %s:\n", i18n.T("templates.actions"))
		fmt.Printf("  P) %s\n", i18n.T("templates.preview_all"))
		fmt.Printf("  T) %s\n", i18n.T("templates.preview_one"))
		fmt.Printf("  N) %s\n", i18n.T("templates.set_default"))
		fmt.Printf("  U) %s\n", i18n.T("templates.edit"))
		fmt.Printf("  Q) %s\n", i18n.T("templates.qr_settings"))
		fmt.Printf("  0) %s\n", i18n.T("prompt.back"))
		fmt.Println()

		choice := strings.ToLower(c.prompt(i18n.T("prompt.choose_option")))

		switch choice {
		case "p":
			c.previewAllTemplates(templates)
		case "t":
			c.previewOneTemplate(templates)
		case "n":
			c.setDefaultTemplate(templates)
		case "u":
			c.editTemplate(templates)
		case "q":
			c.qrSettings()
		case "0":
			return
		}
	}
}

func (c *CLI) selectTemplate(templates []*model.PDFTemplate) *model.PDFTemplate {
	fmt.Println()
	for i, t := range templates {
		fmt.Printf("  %d) %s\n", i+1, t.Name)
	}
	fmt.Println()

	idx := c.promptInt(i18n.T("templates.select_number"), 0) - 1
	if idx < 0 || idx >= len(templates) {
		c.printError(i18n.T("error.invalid_option"))
		return nil
	}
	return templates[idx]
}

func (c *CLI) previewAllTemplates(templates []*model.PDFTemplate) {
	fmt.Println()
	fmt.Println(i18n.T("templates.generating_all"))

	results, err := c.pdfService.GenerateAllPreviews(templates)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	for _, t := range templates {
		if path, ok := results[t.ID]; ok {
			if strings.HasPrefix(path, "error:") {
				fmt.Printf("  %-15s  %s\n", t.Name, path)
			} else {
				fmt.Printf("  %-15s  %s\n", t.Name, path)
				t.PreviewPath = path
				c.templates.Update(t)
			}
		}
	}

	c.printSuccess(i18n.T("templates.all_generated"))

	if c.confirm(i18n.T("templates.open_previews")) {
		for _, t := range templates {
			if t.PreviewPath != "" {
				c.openFile(t.PreviewPath)
			}
		}
	}
	c.waitEnter()
}

func (c *CLI) previewOneTemplate(templates []*model.PDFTemplate) {
	t := c.selectTemplate(templates)
	if t == nil {
		return
	}

	fmt.Println()
	fmt.Printf(i18n.T("templates.generating_one"), t.Name)
	fmt.Println()

	opts := &service.TemplateOptions{
		ShowLogo:  t.ShowLogo,
		ShowQR:    t.ShowQR,
		ShowNotes: t.ShowNotes,
		QRType:    "spayd",
	}

	path, err := c.pdfService.GeneratePreview(t.TemplateCode, opts)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	t.PreviewPath = path
	c.templates.Update(t)

	c.printSuccess(fmt.Sprintf("%s: %s", i18n.T("templates.preview_created"), path))

	if c.confirm(i18n.T("confirm.open_pdf")) {
		c.openFile(path)
	}
	c.waitEnter()
}

func (c *CLI) setDefaultTemplate(templates []*model.PDFTemplate) {
	fmt.Println()
	fmt.Println(i18n.T("templates.choose_default"))

	t := c.selectTemplate(templates)
	if t == nil {
		return
	}

	if err := c.templates.SetDefault(t.ID); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(fmt.Sprintf("%s: %s", i18n.T("templates.default_set"), t.Name))
	c.waitEnter()
}

func (c *CLI) editTemplate(templates []*model.PDFTemplate) {
	t := c.selectTemplate(templates)
	if t == nil {
		return
	}

	fmt.Println()
	fmt.Printf("=== %s: %s ===\n\n", i18n.T("templates.editing"), t.Name)

	t.Name = c.promptDefault(i18n.T("templates.name"), t.Name)

	showLogo := c.promptDefault(i18n.T("templates.show_logo")+" (a/n)", boolToYN(t.ShowLogo))
	t.ShowLogo = showLogo == "a" || showLogo == "y"

	showQR := c.promptDefault(i18n.T("templates.show_qr")+" (a/n)", boolToYN(t.ShowQR))
	t.ShowQR = showQR == "a" || showQR == "y"

	showNotes := c.promptDefault(i18n.T("templates.show_notes")+" (a/n)", boolToYN(t.ShowNotes))
	t.ShowNotes = showNotes == "a" || showNotes == "y"

	if err := c.templates.Update(t); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(i18n.T("templates.updated"))
	c.waitEnter()
}

func (c *CLI) qrSettings() {
	if c.currentSupp == "" {
		c.printError(i18n.T("error.no_supplier"))
		c.waitEnter()
		return
	}

	accounts, err := c.bankAccs.GetBySupplier(c.currentSupp)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}
	if len(accounts) == 0 {
		c.printError(i18n.T("error.no_bank_accounts"))
		c.waitEnter()
		return
	}

	fmt.Println()
	fmt.Printf("=== %s ===\n\n", i18n.T("templates.qr_heading"))

	for i, acc := range accounts {
		fmt.Printf("  %d) %s  [QR: %s]\n", i+1, acc.AccountNumber, acc.QRType)
	}
	fmt.Println()

	idx := c.promptInt(i18n.T("templates.select_account"), 0) - 1
	if idx < 0 || idx >= len(accounts) {
		return
	}
	acc := accounts[idx]

	fmt.Println()
	fmt.Printf("  1) SPAYD / QR Platba  (%s)\n", i18n.T("templates.qr_spayd_desc"))
	fmt.Printf("  2) Pay by Square      (%s)\n", i18n.T("templates.qr_pbs_desc"))
	fmt.Printf("  3) EPC QR / GiroCode  (%s)\n", i18n.T("templates.qr_epc_desc"))
	fmt.Printf("  4) %s\n", i18n.T("templates.qr_none"))
	fmt.Println()

	qrChoice := c.promptInt(i18n.T("templates.select_qr_type"), 0)
	switch qrChoice {
	case 1:
		acc.QRType = "spayd"
	case 2:
		acc.QRType = "pay_by_square"
	case 3:
		acc.QRType = "epc"
	case 4:
		acc.QRType = "none"
	default:
		return
	}

	if err := c.bankAccs.Update(acc); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(fmt.Sprintf("%s: %s → %s", i18n.T("templates.qr_updated"), acc.AccountNumber, acc.QRType))
	c.waitEnter()
}

func boolToYN(b bool) string {
	if b {
		return "a"
	}
	return "n"
}
