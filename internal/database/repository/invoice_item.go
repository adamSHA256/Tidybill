package repository

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type InvoiceItemRepository struct {
	db *sql.DB
}

func NewInvoiceItemRepository(db *sql.DB) *InvoiceItemRepository {
	return &InvoiceItemRepository{db: db}
}

func (r *InvoiceItemRepository) Create(item *model.InvoiceItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.Calculate()

	// Convert empty string to NULL for foreign key
	var itemID interface{}
	if item.ItemID == "" {
		itemID = nil
	} else {
		itemID = item.ItemID
	}

	_, err := r.db.Exec(`
		INSERT INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.InvoiceID, itemID, item.Description, item.Quantity, item.Unit,
		item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount, item.Total, item.Position)
	return err
}

func (r *InvoiceItemRepository) GetByInvoice(invoiceID string) ([]model.InvoiceItem, error) {
	rows, err := r.db.Query(`
		SELECT id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position
		FROM invoice_items WHERE invoice_id = ? ORDER BY position ASC`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InvoiceItem
	for rows.Next() {
		item := model.InvoiceItem{}
		var itemID sql.NullString
		if err := rows.Scan(&item.ID, &item.InvoiceID, &itemID, &item.Description, &item.Quantity,
			&item.Unit, &item.UnitPrice, &item.VATRate, &item.Subtotal, &item.VATAmount,
			&item.Total, &item.Position); err != nil {
			return nil, err
		}
		if itemID.Valid {
			item.ItemID = itemID.String
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *InvoiceItemRepository) DeleteByInvoice(invoiceID string) error {
	_, err := r.db.Exec("DELETE FROM invoice_items WHERE invoice_id = ?", invoiceID)
	return err
}

func (r *InvoiceItemRepository) CreateBatch(items []model.InvoiceItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO invoice_items (id, invoice_id, item_id, description, quantity, unit,
			unit_price, vat_rate, subtotal, vat_amount, total, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range items {
		item := &items[i]
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.Calculate()

		// Convert empty string to NULL for foreign key
		var itemID interface{}
		if item.ItemID == "" {
			itemID = nil
		} else {
			itemID = item.ItemID
		}

		if _, err := stmt.Exec(item.ID, item.InvoiceID, itemID, item.Description, item.Quantity,
			item.Unit, item.UnitPrice, item.VATRate, item.Subtotal, item.VATAmount,
			item.Total, item.Position); err != nil {
			return err
		}
	}

	return tx.Commit()
}
