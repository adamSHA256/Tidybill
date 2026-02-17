package cli

import (
	"fmt"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/adamSHA256/tidybill/internal/model"
)

const itemsPerPage = 20

func (c *CLI) itemsMenu() {
	offset := 0
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.T("heading.items_catalog"))
		fmt.Println()

		total, _ := c.items.Count()
		items, err := c.items.List(itemsPerPage, offset)
		if err != nil {
			c.printError(err.Error())
			c.waitEnter()
			return
		}

		if total == 0 {
			fmt.Println("  " + i18n.T("info.no_items"))
		} else {
			for i, item := range items {
				num := offset + i + 1
				cat := ""
				if item.Category != "" {
					cat = fmt.Sprintf(" [%s]", item.Category)
				}
				fmt.Printf("  %d) %s%s — %.2f %s\n",
					num, item.Description, cat, item.DefaultPrice, item.DefaultUnit)
			}

			page := (offset / itemsPerPage) + 1
			totalPages := (total + itemsPerPage - 1) / itemsPerPage
			if totalPages > 1 {
				fmt.Println()
				fmt.Printf("  %s %d/%d\n", i18n.T("label.page"), page, totalPages)
				if page < totalPages {
					fmt.Println("  " + i18n.T("action.next_page"))
				}
				if page > 1 {
					fmt.Println("  " + i18n.T("action.prev_page"))
				}
			}
		}

		fmt.Println()
		fmt.Println("  " + i18n.T("action.new_item_catalog"))
		fmt.Println("  " + i18n.T("action.search_items"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "0", "":
			return
		case "n":
			c.createItem()
		case "h":
			c.searchItems()
		case ">":
			if offset+itemsPerPage < total {
				offset += itemsPerPage
			}
		case "<":
			if offset-itemsPerPage >= 0 {
				offset -= itemsPerPage
			}
		default:
			idx := 0
			fmt.Sscanf(choice, "%d", &idx)
			sliceIdx := idx - 1 - offset
			if sliceIdx >= 0 && sliceIdx < len(items) {
				c.editItem(items[sliceIdx])
			}
		}
	}
}

func (c *CLI) createItem() *model.Item {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.new_item"))
	fmt.Println(i18n.T("prompt.enter_0_back"))
	fmt.Println()

	item := model.NewItem()

	desc, goBack := c.promptMaxLenWithBack(i18n.T("prompt.item_description"), model.MaxDescriptionLen)
	if goBack {
		return nil
	}
	if desc == "" {
		c.printError(i18n.T("error.description_required"))
		c.waitEnter()
		return nil
	}

	existing, _ := c.items.FindByDescription(desc)
	if existing != nil {
		fmt.Println()
		fmt.Printf("  %s\n", i18n.Tf("warning.item_exists", existing.Description, existing.DefaultPrice))
		if !c.confirm(i18n.T("confirm.create_duplicate")) {
			return existing
		}
	}

	item.Description = desc

	// TODO: Dynamic VAT rates from settings (GET /api/vat-rates) are not implemented in CLI.
	// The GUI version manages custom VAT rates and a default rate via settings.
	// CLI still uses hardcoded 21% default for VAT payers.
	defaultVAT := 0.0
	supplier, _ := c.suppliers.GetByID(c.currentSupp)
	if supplier != nil && supplier.IsVATPayer {
		defaultVAT = 21.0
	}

	item.DefaultPrice = c.promptFloat(i18n.T("prompt.default_price"), 0)
	item.DefaultUnit = c.selectUnit("")
	item.DefaultVATRate = c.promptFloat(i18n.T("prompt.default_vat"), defaultVAT)
	item.Category = c.promptCategory()

	if err := c.items.Create(item); err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return nil
	}

	c.printSuccess(i18n.T("success.item_created"))
	c.waitEnter()
	return item
}

func (c *CLI) promptCategory() string {
	categories, _ := c.items.GetExistingCategories()

	if len(categories) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("prompt.select_category"))
		for i, cat := range categories {
			fmt.Printf("  %d) %s\n", i+1, cat)
		}
		fmt.Println("  " + i18n.T("action.new_category"))
		fmt.Println()
	}

	input := c.promptMaxLen(i18n.T("prompt.category"), model.MaxCategoryLen)

	if len(categories) > 0 {
		idx := 0
		fmt.Sscanf(input, "%d", &idx)
		if idx > 0 && idx <= len(categories) {
			return categories[idx-1]
		}
	}

	return model.NormalizeCategory(input)
}

func (c *CLI) editItem(item *model.Item) {
	for {
		c.clearScreen()
		fmt.Printf("=== %s ===\n", i18n.Tf("heading.item_detail", item.Description))
		fmt.Println()

		fmt.Printf("  %s: %s\n", i18n.T("label.description"), item.Description)
		fmt.Printf("  %s: %.2f\n", i18n.T("label.default_price"), item.DefaultPrice)
		fmt.Printf("  %s: %s\n", i18n.T("label.default_unit"), item.DefaultUnit)
		fmt.Printf("  %s: %.1f%%\n", i18n.T("label.default_vat"), item.DefaultVATRate)
		if item.Category != "" {
			fmt.Printf("  %s: %s\n", i18n.T("label.category"), item.Category)
		}
		fmt.Println()
		fmt.Printf("  %s: %d\n", i18n.T("label.usage_count"), item.UsageCount)
		if item.LastUsedPrice > 0 {
			fmt.Printf("  %s: %.2f\n", i18n.T("label.last_price"), item.LastUsedPrice)
		}
		if item.LastCustomerID != "" {
			cust, _ := c.customers.GetByID(item.LastCustomerID)
			if cust != nil {
				fmt.Printf("  %s: %s\n", i18n.T("label.last_customer"), cust.Name)
			} else {
				fmt.Printf("  %s: %s\n", i18n.T("label.last_customer"), i18n.T("label.deleted_customer"))
			}
		}
		fmt.Println()

		fmt.Println("  " + i18n.T("action.edit_details"))
		fmt.Println("  " + i18n.T("action.delete_item"))
		fmt.Println("  " + i18n.T("action.back"))
		fmt.Println()

		choice := c.prompt(i18n.T("prompt.choose_option"))

		switch strings.ToLower(choice) {
		case "0", "":
			return
		case "e":
			c.editItemDetails(item)
		case "x":
			if c.confirm(i18n.T("confirm.delete_item")) {
				if err := c.items.Delete(item.ID); err != nil {
					c.printError(err.Error())
					c.waitEnter()
				} else {
					c.printSuccess(i18n.T("success.item_deleted"))
					c.waitEnter()
					return
				}
			}
		}
	}
}

func (c *CLI) editItemDetails(item *model.Item) {
	c.clearScreen()
	fmt.Printf("=== %s ===\n", i18n.T("heading.edit_item"))
	fmt.Println(i18n.T("prompt.leave_empty_hint"))
	fmt.Println()

	item.Description = c.promptDefaultMaxLen(i18n.T("prompt.item_description"), item.Description, model.MaxDescriptionLen)
	item.DefaultPrice = c.promptFloat(i18n.T("prompt.default_price"), item.DefaultPrice)
	item.DefaultUnit = c.selectUnit(item.DefaultUnit)
	item.DefaultVATRate = c.promptFloat(i18n.T("prompt.default_vat"), item.DefaultVATRate)
	item.Category = c.promptCategory()

	if err := c.items.Update(item); err != nil {
		c.printError(err.Error())
	} else {
		c.printSuccess(i18n.T("success.item_updated"))
	}
	c.waitEnter()
}

func (c *CLI) searchItems() {
	query := c.prompt(i18n.T("prompt.search_items"))
	if query == "" {
		return
	}

	items, err := c.items.Search(query)
	if err != nil {
		c.printError(err.Error())
		c.waitEnter()
		return
	}

	if len(items) == 0 {
		fmt.Println(i18n.T("info.nothing_found"))
		c.waitEnter()
		return
	}

	fmt.Println()
	for i, item := range items {
		cat := ""
		if item.Category != "" {
			cat = fmt.Sprintf(" [%s]", item.Category)
		}
		fmt.Printf("  %d) %s%s — %.2f %s\n",
			i+1, item.Description, cat, item.DefaultPrice, item.DefaultUnit)
	}

	fmt.Println()
	choice := c.prompt(i18n.T("prompt.select_for_detail"))
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx > 0 && idx <= len(items) {
		c.editItem(items[idx-1])
	}
}
