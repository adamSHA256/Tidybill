package cli

import (
	"fmt"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (c *CLI) customersMenu() {
	for {
		c.clearScreen()
		fmt.Println("=== ODBĚRATELÉ ===")
		fmt.Println()

		customers, err := c.customers.List()
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		if len(customers) == 0 {
			fmt.Println("  Zatím nemáte žádné odběratele.")
		} else {
			for i, cust := range customers {
				fmt.Printf("  %d) %s", i+1, cust.Name)
				if cust.ICO != "" {
					fmt.Printf(" (IČO: %s)", cust.ICO)
				}
				fmt.Println()
			}
		}

		fmt.Println()
		fmt.Println("  N) Nový odběratel")
		fmt.Println("  H) Hledat")
		fmt.Println("  0) Zpět")
		fmt.Println()

		choice := c.prompt("Vyberte možnost")

		switch choice {
		case "0", "":
			return
		case "n", "N":
			c.createCustomer()
		case "h", "H":
			c.searchCustomers()
		default:
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			if idx > 0 && idx <= len(customers) {
				c.editCustomer(customers[idx-1])
			}
		}
	}
}

func (c *CLI) createCustomer() *model.Customer {
	c.clearScreen()
	fmt.Println("=== NOVÝ ODBĚRATEL ===")
	fmt.Println("(Zadejte 0 pro návrat zpět)")
	fmt.Println()

	cust := model.NewCustomer()

	name, goBack := c.promptWithBack("Název firmy / Jméno")
	if goBack {
		return nil
	}
	if name == "" {
		c.printError("Název je povinný")
		c.waitEnter()
		return nil
	}
	cust.Name = name

	cust.Street = c.prompt("Ulice a číslo")
	cust.City = c.prompt("Město")
	cust.ZIP = c.prompt("PSČ")
	cust.Region = c.prompt("Kraj (volitelné)")
	cust.Country = c.promptDefault("Země", "CZ")
	cust.ICO = c.prompt("IČO")
	cust.DIC = c.prompt("DIČ")
	cust.Email = c.prompt("E-mail")
	cust.Phone = c.prompt("Telefon")
	cust.DefaultDueDays = c.promptInt("Výchozí splatnost (dny)", 14)
	cust.Notes = c.prompt("Poznámky")

	if err := c.customers.Create(cust); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return nil
	}

	c.printSuccess("Odběratel byl vytvořen")
	c.waitEnter()
	return cust
}

func (c *CLI) editCustomer(cust *model.Customer) {
	for {
		c.clearScreen()
		fmt.Printf("=== ODBĚRATEL: %s ===\n", cust.Name)
		fmt.Println()
		fmt.Printf("  Adresa:    %s, %s %s, %s\n", cust.Street, cust.ZIP, cust.City, cust.Country)
		if cust.Region != "" {
			fmt.Printf("  Kraj:      %s\n", cust.Region)
		}
		fmt.Printf("  IČO:       %s\n", cust.ICO)
		fmt.Printf("  DIČ:       %s\n", cust.DIC)
		fmt.Printf("  E-mail:    %s\n", cust.Email)
		fmt.Printf("  Telefon:   %s\n", cust.Phone)
		fmt.Printf("  Splatnost: %d dní\n", cust.DefaultDueDays)
		if cust.Notes != "" {
			fmt.Printf("  Poznámky:  %s\n", cust.Notes)
		}
		fmt.Println()

		fmt.Println("  E) Upravit údaje")
		fmt.Println("  F) Zobrazit faktury")
		fmt.Println("  X) Smazat odběratele")
		fmt.Println("  0) Zpět")
		fmt.Println()

		choice := c.prompt("Vyberte možnost")

		switch choice {
		case "0", "":
			return
		case "e", "E":
			c.editCustomerDetails(cust)
		case "f", "F":
			c.listCustomerInvoices(cust)
		case "x", "X":
			if c.confirm("Opravdu smazat odběratele?") {
				if err := c.customers.Delete(cust.ID); err != nil {
					c.printError(err.Error())
					c.waitEnter()
				} else {
					c.printSuccess("Odběratel byl smazán")
					c.waitEnter()
					return
				}
			}
		}
	}
}

func (c *CLI) editCustomerDetails(cust *model.Customer) {
	c.clearScreen()
	fmt.Println("=== ÚPRAVA ODBĚRATELE ===")
	fmt.Println("(Nechte prázdné pro zachování stávající hodnoty)")
	fmt.Println()

	cust.Name = c.promptDefault("Název", cust.Name)
	cust.Street = c.promptDefault("Ulice", cust.Street)
	cust.City = c.promptDefault("Město", cust.City)
	cust.ZIP = c.promptDefault("PSČ", cust.ZIP)
	cust.Region = c.promptDefault("Kraj", cust.Region)
	cust.Country = c.promptDefault("Země", cust.Country)
	cust.ICO = c.promptDefault("IČO", cust.ICO)
	cust.DIC = c.promptDefault("DIČ", cust.DIC)
	cust.Email = c.promptDefault("E-mail", cust.Email)
	cust.Phone = c.promptDefault("Telefon", cust.Phone)
	cust.DefaultDueDays = c.promptInt("Splatnost (dny)", cust.DefaultDueDays)
	cust.Notes = c.promptDefault("Poznámky", cust.Notes)

	if err := c.customers.Update(cust); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess("Odběratel byl aktualizován")
	}
	c.waitEnter()
}

func (c *CLI) searchCustomers() {
	query := c.prompt("Hledat (název nebo IČO)")
	if query == "" {
		return
	}

	customers, err := c.customers.Search(query)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(customers) == 0 {
		fmt.Println("Nic nenalezeno.")
		c.waitEnter()
		return
	}

	fmt.Println()
	for i, cust := range customers {
		fmt.Printf("  %d) %s", i+1, cust.Name)
		if cust.ICO != "" {
			fmt.Printf(" (IČO: %s)", cust.ICO)
		}
		fmt.Println()
	}

	fmt.Println()
	choice := c.prompt("Vyberte číslo pro detail (0 = zpět)")
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(customers) {
		c.editCustomer(customers[idx-1])
	}
}

func (c *CLI) listCustomerInvoices(cust *model.Customer) {
	invoices, err := c.invoices.List("", cust.ID)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	c.clearScreen()
	fmt.Printf("=== FAKTURY: %s ===\n", cust.Name)
	fmt.Println()

	if len(invoices) == 0 {
		fmt.Println("Žádné faktury.")
	} else {
		for _, inv := range invoices {
			fmt.Printf("  %s | %s | %.2f %s | %s\n",
				inv.InvoiceNumber,
				inv.IssueDate.Format("02.01.2006"),
				inv.Total,
				inv.Currency,
				inv.Status)
		}
	}

	c.waitEnter()
}

func (c *CLI) selectCustomer() *model.Customer {
	cust, _ := c.selectCustomerWithBack()
	return cust
}

func (c *CLI) selectCustomerWithBack() (*model.Customer, bool) {
	customers, _ := c.customers.List()

	if len(customers) == 0 {
		fmt.Println("Nemáte žádné odběratele. Vytvořte nového:")
		cust := c.createCustomer()
		return cust, cust == nil
	}

	fmt.Println()
	fmt.Println("Vyberte odběratele:")
	for i, cust := range customers {
		fmt.Printf("  %d) %s\n", i+1, cust.Name)
	}
	fmt.Println("  N) Nový odběratel")
	fmt.Println("  0) Zpět")
	fmt.Println()

	choice := c.prompt("Volba")

	if choice == "0" {
		return nil, true
	}

	if choice == "n" || choice == "N" {
		cust := c.createCustomer()
		return cust, cust == nil
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(customers) {
		return customers[idx-1], false
	}

	return nil, false
}
