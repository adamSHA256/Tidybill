package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

// Placeholder menus - will be implemented later

func (c *CLI) syncMenu() {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.sync"))
	fmt.Println()
	fmt.Println(i18n.T("info.feature_coming"))
	c.waitEnter()
}

