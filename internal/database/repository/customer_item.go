package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type CustomerItemRepository struct {
	db DBTX
}

func NewCustomerItemRepository(db DBTX) *CustomerItemRepository {
	return &CustomerItemRepository{db: db}
}

func (r *CustomerItemRepository) WithDB(db DBTX) *CustomerItemRepository {
	return &CustomerItemRepository{db: db}
}

func (r *CustomerItemRepository) Upsert(customerID, itemID string, price, quantity float64) error {
	now := time.Now()
	id := uuid.New().String()

	_, err := r.db.Exec(`
		INSERT INTO customer_items (id, customer_id, item_id, last_price, last_quantity, usage_count, last_used_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(customer_id, item_id) DO UPDATE SET
			last_price = excluded.last_price,
			last_quantity = excluded.last_quantity,
			usage_count = usage_count + 1,
			last_used_at = excluded.last_used_at`,
		id, customerID, itemID, price, quantity, now)
	return err
}

func (r *CustomerItemRepository) GetByCustomer(customerID string) ([]*model.CustomerItem, error) {
	rows, err := r.db.Query(`
		SELECT ci.id, ci.customer_id, ci.item_id, ci.last_price, ci.last_quantity,
			ci.usage_count, ci.last_used_at,
			i.description, i.category, i.default_unit, i.default_vat_rate
		FROM customer_items ci
		JOIN items i ON ci.item_id = i.id
		WHERE ci.customer_id = ?
		ORDER BY ci.usage_count DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.CustomerItem
	for rows.Next() {
		ci := &model.CustomerItem{}
		var lastUsedAt sql.NullTime
		var category sql.NullString
		if err := rows.Scan(&ci.ID, &ci.CustomerID, &ci.ItemID, &ci.LastPrice,
			&ci.LastQuantity, &ci.UsageCount, &lastUsedAt,
			&ci.ItemDescription, &category, &ci.ItemDefaultUnit, &ci.ItemDefaultVAT); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			ci.LastUsedAt = lastUsedAt.Time
		}
		if category.Valid {
			ci.ItemCategory = category.String
		}
		items = append(items, ci)
	}
	return items, rows.Err()
}

func (r *CustomerItemRepository) GetByCustomerAndItem(customerID, itemID string) (*model.CustomerItem, error) {
	ci := &model.CustomerItem{}
	var lastUsedAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, customer_id, item_id, last_price, last_quantity, usage_count, last_used_at
		FROM customer_items WHERE customer_id = ? AND item_id = ?`,
		customerID, itemID).Scan(
		&ci.ID, &ci.CustomerID, &ci.ItemID, &ci.LastPrice,
		&ci.LastQuantity, &ci.UsageCount, &lastUsedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		ci.LastUsedAt = lastUsedAt.Time
	}
	return ci, nil
}
