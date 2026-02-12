package cli

import (
	"fmt"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

func (c *CLI) settingsMenu() {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.settings"))
		fmt.Println()

		currentLang := i18n.GetLang()
		fmt.Printf("  %s %s\n\n", i18n.T("settings.current_language"), langName(currentLang))

		fmt.Printf("  L) %s\n", i18n.T("settings.change_language"))
		fmt.Printf("  %s\n", i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "l":
			c.changeLanguage()
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
