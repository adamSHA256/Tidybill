package cli

import "fmt"

// Placeholder menus for Phase 1 - will be implemented later

func (c *CLI) itemsMenu() {
	c.clearScreen()
	fmt.Println("=== KATALOG POLOŽEK ===")
	fmt.Println()
	fmt.Println("Tato funkce bude dostupná v další verzi.")
	c.waitEnter()
}

func (c *CLI) syncMenu() {
	c.clearScreen()
	fmt.Println("=== SYNC / IMPORT / EXPORT ===")
	fmt.Println()
	fmt.Println("Tato funkce bude dostupná v další verzi.")
	c.waitEnter()
}

func (c *CLI) templatesMenu() {
	c.clearScreen()
	fmt.Println("=== ŠABLONY PDF ===")
	fmt.Println()
	fmt.Println("Tato funkce bude dostupná v další verzi.")
	c.waitEnter()
}

func (c *CLI) settingsMenu() {
	c.clearScreen()
	fmt.Println("=== NASTAVENÍ ===")
	fmt.Println()
	fmt.Println("Tato funkce bude dostupná v další verzi.")
	c.waitEnter()
}
