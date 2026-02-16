package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type cliUnit struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
}

func (c *CLI) settingsMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.settings"))
		fmt.Println()

		currentLang := i18n.GetLang()
		fmt.Printf("  %s %s\n\n", i18n.T("settings.current_language"), langName(currentLang))

		fmt.Printf("  L) %s\n", i18n.T("settings.change_language"))
		fmt.Printf("  %s\n", i18n.T("action.change_directories"))
		fmt.Printf("  U) %s\n", i18n.T("settings.manage_units"))
		fmt.Printf("  %s\n", i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "l":
			c.changeLanguage()
		case "d":
			c.changeDirectories()
		case "u":
			c.manageUnits()
		case "0", "q":
			return
		}
	}
}

func (c *CLI) changeLanguage() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n\n", i18n.T("settings.change_language"))

	langs := i18n.AvailableLanguages()
	currentLang := i18n.GetLang()

	for idx, lang := range langs {
		marker := "  "
		if lang == currentLang {
			marker = "* "
		}
		fmt.Printf("  %s%d) %s\n", marker, idx+1, langName(lang))
	}
	fmt.Printf("\n  %s\n\n", i18n.T("action.back"))

	choice := c.prompt(i18n.T("prompt.choose_option"))

	var newLang i18n.Lang
	switch choice {
	case "1":
		newLang = i18n.CS
	case "2":
		newLang = i18n.SK
	case "3":
		newLang = i18n.EN
	default:
		return
	}

	i18n.SetLang(newLang)
	c.settings.Set("language", string(newLang))

	c.printSuccess(i18n.Tf("success.language_changed", langName(newLang)))
	c.waitEnter()
}

func (c *CLI) changeDirectories() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n\n", i18n.T("heading.directories"))

	fmt.Printf("  "+i18n.T("label.dir_logos")+"\n", c.cfg.LogoDir)
	fmt.Printf("  "+i18n.T("label.dir_pdfs")+"\n", c.cfg.PDFDir)
	fmt.Printf("  "+i18n.T("label.dir_previews")+"\n", c.cfg.PreviewDir)
	fmt.Println()
	fmt.Println(i18n.T("prompt.leave_empty_hint"))
	fmt.Println()

	newLogoDir := c.promptDefault(i18n.T("prompt.dir_logos"), c.cfg.LogoDir)
	newPDFDir := c.promptDefault(i18n.T("prompt.dir_pdfs"), c.cfg.PDFDir)
	newPreviewDir := c.promptDefault(i18n.T("prompt.dir_previews"), c.cfg.PreviewDir)

	// Validate and create directories
	for _, dir := range []string{newLogoDir, newPDFDir, newPreviewDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			c.printError(i18n.Tf("error.directory_invalid", err))
			c.waitEnter()
			return
		}
	}

	// Save to settings DB
	dirSettings := map[string]string{
		"dir.logos":    newLogoDir,
		"dir.pdfs":     newPDFDir,
		"dir.previews": newPreviewDir,
	}
	for key, val := range dirSettings {
		if err := c.settings.Set(key, val); err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}
	}

	// Update in-memory config
	c.cfg.LogoDir = newLogoDir
	c.cfg.PDFDir = newPDFDir
	c.cfg.PreviewDir = newPreviewDir

	c.printSuccess(i18n.T("success.directories_updated"))
	c.waitEnter()
}

// selectUnit shows available units from settings as a numbered list
// and allows the user to pick one or type a custom name.
// preferredDefault overrides the is_default from settings when non-empty.
func (c *CLI) selectUnit(preferredDefault string) string {
	units := c.loadUnits()

	// Determine which unit to pre-select
	defaultName := ""
	for _, u := range units {
		if preferredDefault != "" && u.Name == preferredDefault {
			defaultName = preferredDefault
			break
		}
		if u.IsDefault && defaultName == "" {
			defaultName = u.Name
		}
	}
	if preferredDefault != "" && defaultName == "" {
		// preferredDefault not in the list — still use it as the fallback
		defaultName = preferredDefault
	}
	if defaultName == "" && len(units) > 0 {
		defaultName = units[0].Name
	}

	fmt.Println()
	fmt.Printf("  %s:\n", i18n.T("prompt.unit"))
	for i, u := range units {
		marker := "  "
		if u.Name == defaultName {
			marker = "* "
		}
		fmt.Printf("  %s%d) %s\n", marker, i+1, u.Name)
	}
	fmt.Printf("  %s\n", i18n.T("settings.add_unit_or_type"))
	fmt.Println()

	input := c.promptDefault(i18n.T("prompt.choice"), "")
	if input == "" {
		return defaultName
	}

	idx := 0
	fmt.Sscanf(input, "%d", &idx)
	if idx > 0 && idx <= len(units) {
		return units[idx-1].Name
	}

	// Treat as custom unit name
	return input
}

func (c *CLI) loadUnits() []cliUnit {
	raw, err := c.settings.Get("units")
	if err != nil || raw == "" {
		return []cliUnit{
			{Name: "ks", IsDefault: true},
			{Name: "hod"},
			{Name: "den"},
			{Name: "m\u00B2"},
		}
	}
	var units []cliUnit
	if err := json.Unmarshal([]byte(raw), &units); err != nil {
		return []cliUnit{{Name: "ks", IsDefault: true}}
	}
	return units
}

func (c *CLI) saveUnits(units []cliUnit) error {
	data, err := json.Marshal(units)
	if err != nil {
		return err
	}
	return c.settings.Set("units", string(data))
}

func (c *CLI) manageUnits() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n\n", i18n.T("settings.units_title"))

		units := c.loadUnits()
		for i, u := range units {
			def := ""
			if u.IsDefault {
				def = " " + i18n.T("label.default_upper")
			}
			fmt.Printf("  %d) %s%s\n", i+1, u.Name, def)
		}
		fmt.Println()

		fmt.Printf("  A) %s\n", i18n.T("settings.add_unit"))
		fmt.Printf("  R) %s\n", i18n.T("settings.remove_unit"))
		fmt.Printf("  D) %s\n", i18n.T("settings.set_default_unit"))
		fmt.Printf("  %s\n\n", i18n.T("action.back"))

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "a":
			name := c.prompt(i18n.T("prompt.unit"))
			if name == "" || name == "0" {
				continue
			}
			units = append(units, cliUnit{Name: name})
			if err := c.saveUnits(units); err != nil {
				c.printError(err.Error())
			} else {
				c.printSuccess(i18n.T("success.units_updated"))
			}
			c.waitEnter()

		case "r":
			if len(units) <= 1 {
				c.printError(i18n.T("error.invalid_option"))
				c.waitEnter()
				continue
			}
			numStr := c.prompt(i18n.T("prompt.choose_option"))
			idx := 0
			fmt.Sscanf(numStr, "%d", &idx)
			idx--
			if idx < 0 || idx >= len(units) {
				continue
			}
			units = append(units[:idx], units[idx+1:]...)
			// Ensure at least one default
			hasDefault := false
			for _, u := range units {
				if u.IsDefault {
					hasDefault = true
				}
			}
			if !hasDefault && len(units) > 0 {
				units[0].IsDefault = true
			}
			if err := c.saveUnits(units); err != nil {
				c.printError(err.Error())
			} else {
				c.printSuccess(i18n.T("success.units_updated"))
			}
			c.waitEnter()

		case "d":
			numStr := c.prompt(i18n.T("prompt.choose_option"))
			idx := 0
			fmt.Sscanf(numStr, "%d", &idx)
			idx--
			if idx < 0 || idx >= len(units) {
				continue
			}
			for i := range units {
				units[i].IsDefault = (i == idx)
			}
			if err := c.saveUnits(units); err != nil {
				c.printError(err.Error())
			} else {
				c.printSuccess(i18n.T("success.units_updated"))
			}
			c.waitEnter()

		case "0", "q":
			return
		}
	}
}

func langName(lang i18n.Lang) string {
	switch lang {
	case i18n.CS:
		return "Čeština"
	case i18n.SK:
		return "Slovenčina"
	case i18n.EN:
		return "English"
	default:
		return string(lang)
	}
}
