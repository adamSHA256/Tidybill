package cli

import (
	"fmt"
	"io"
	"os"
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
		fmt.Printf("  D) %s\n", i18n.T("templates.duplicate"))
		fmt.Printf("  E) %s\n", i18n.T("templates.export_yaml"))
		fmt.Printf("  I) %s\n", i18n.T("templates.import_yaml"))
		fmt.Printf("  X) %s\n", i18n.T("templates.delete_custom"))
		fmt.Printf("  A) %s\n", i18n.T("templates.show_ai_prompt"))
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
		case "d":
			c.duplicateTemplate(templates)
		case "e":
			c.exportTemplateYAML(templates)
		case "i":
			c.importTemplateYAML()
		case "x":
			c.deleteCustomTemplate(templates)
		case "a":
			c.showAIPrompt()
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

	yamlSource := t.YAMLSource
	if t.IsBuiltin {
		yamlSource = ""
	}
	path, err := c.pdfService.GeneratePreviewWithYAML(t.TemplateCode, yamlSource, opts)
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

func (c *CLI) duplicateTemplate(templates []*model.PDFTemplate) {
	fmt.Println()
	fmt.Println("  " + i18n.T("templates.select_to_duplicate"))
	t := c.selectTemplate(templates)
	if t == nil {
		return
	}

	name := c.prompt(i18n.T("templates.new_name"))
	if name == "" {
		return
	}

	newTmpl, err := c.templates.Duplicate(t.ID, name)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	// Built-in templates don't store YAML in DB - inject it into the new custom copy
	if t.IsBuiltin {
		yamlSrc := service.GetBuiltinYAML(t.TemplateCode)
		if yamlSrc != "" {
			if err := c.templates.UpdateYAMLSource(newTmpl.ID, yamlSrc); err != nil {
				c.printError(err.Error())
				c.waitEnter()
				return
			}
		}
	}

	c.printSuccess(fmt.Sprintf(i18n.T("templates.duplicate_created"), name, newTmpl.ID))
	c.waitEnter()
}

func (c *CLI) exportTemplateYAML(templates []*model.PDFTemplate) {
	fmt.Println()
	fmt.Println("  " + i18n.T("templates.select_to_export"))
	t := c.selectTemplate(templates)
	if t == nil {
		return
	}

	yamlSource := t.YAMLSource
	if t.IsBuiltin && yamlSource == "" {
		yamlSource = service.GetBuiltinYAML(t.TemplateCode)
	}

	if yamlSource == "" {
		c.printError(i18n.T("templates.no_yaml_source"))
		c.waitEnter()
		return
	}

	fmt.Println()
	fmt.Println("--- YAML START ---")
	fmt.Println(yamlSource)
	fmt.Println("--- YAML END ---")
	fmt.Println()

	// Option to save to file
	filePath := c.promptDefault(i18n.T("templates.save_to_file"), "")
	if filePath != "" {
		if err := os.WriteFile(filePath, []byte(yamlSource), 0644); err != nil {
			c.printError(err.Error())
		} else {
			c.printSuccess(fmt.Sprintf("%s: %s", i18n.T("templates.file_saved"), filePath))
		}
	}
	c.waitEnter()
}

func (c *CLI) importTemplateYAML() {
	fmt.Println()
	name := c.prompt(i18n.T("templates.new_name"))
	if name == "" {
		return
	}

	filePath := c.prompt(i18n.T("templates.yaml_file_path"))

	var yamlSource string

	if filePath == "stdin" {
		fmt.Println(i18n.T("templates.paste_yaml"))
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}
		yamlSource = string(data)
	} else {
		data, err := os.ReadFile(filePath)
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}
		yamlSource = string(data)
	}

	// Validate
	if err := service.ValidateYAML(yamlSource); err != nil {
		c.printError(i18n.T("templates.invalid_yaml") + ": " + err.Error())
		c.waitEnter()
		return
	}

	// Get default template as parent for the duplicate
	defaultTmpl, err := c.templates.GetDefault()
	if err != nil || defaultTmpl == nil {
		c.printError(i18n.T("templates.no_default_found"))
		c.waitEnter()
		return
	}

	newTmpl, err := c.templates.Duplicate(defaultTmpl.ID, name)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if err := c.templates.UpdateYAMLSource(newTmpl.ID, yamlSource); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(fmt.Sprintf(i18n.T("templates.import_created"), name, newTmpl.ID))
	c.waitEnter()
}

func (c *CLI) deleteCustomTemplate(templates []*model.PDFTemplate) {
	// Filter to show only custom templates
	var custom []*model.PDFTemplate
	for _, t := range templates {
		if !t.IsBuiltin {
			custom = append(custom, t)
		}
	}

	if len(custom) == 0 {
		fmt.Println()
		fmt.Println("  " + i18n.T("templates.no_custom_templates"))
		c.waitEnter()
		return
	}

	fmt.Println()
	fmt.Println("  " + i18n.T("templates.custom_templates"))
	t := c.selectTemplate(custom)
	if t == nil {
		return
	}

	if !c.confirm(fmt.Sprintf(i18n.T("templates.confirm_delete"), t.Name)) {
		return
	}

	if err := c.templates.Delete(t.ID); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.printSuccess(fmt.Sprintf(i18n.T("templates.template_deleted"), t.Name))
	c.waitEnter()
}

func (c *CLI) showAIPrompt() {
	fmt.Println()
	fmt.Println(service.TemplateEditingAIPrompt)
	c.waitEnter()
}

func boolToYN(b bool) string {
	if b {
		return "a"
	}
	return "n"
}
