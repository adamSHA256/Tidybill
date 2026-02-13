package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type ItemRepository struct {
	db DBTX
}

func NewItemRepository(db DBTX) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) WithDB(db DBTX) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(item *model.Item) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	item.Category = model.NormalizeCategory(item.Category)

	_, err := r.db.Exec(`
		INSERT INTO items (id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
		item.Category, item.LastUsedPrice, nullString(item.LastCustomerID),
		item.UsageCount, item.CreatedAt, item.UpdatedAt)
	return err
}

func (r *ItemRepository) Update(item *model.Item) error {
	item.UpdatedAt = time.Now()
	item.Category = model.NormalizeCategory(item.Category)

	_, err := r.db.Exec(`
		UPDATE items SET description=?, default_price=?, default_unit=?, default_vat_rate=?,
			category=?, updated_at=?
		WHERE id=?`,
		item.Description, item.DefaultPrice, item.DefaultUnit, item.DefaultVATRate,
		item.Category, item.UpdatedAt, item.ID)
	return err
}

func (r *ItemRepository) GetByID(id string) (*model.Item, error) {
	item := &model.Item{}
	var lastCustID sql.NullString
	err := r.db.QueryRow(`
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items WHERE id = ?`, id).Scan(
		&item.ID, &item.Description, &item.DefaultPrice, &item.DefaultUnit, &item.DefaultVATRate,
		&item.Category, &item.LastUsedPrice, &lastCustID,
		&item.UsageCount, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastCustID.Valid {
		item.LastCustomerID = lastCustID.String
	}
	return item, nil
}

func (r *ItemRepository) List(limit, offset int) ([]*model.Item, error) {
	query := `
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items ORDER BY usage_count DESC, description ASC`

	var args []interface{}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanItems(rows)
}

func (r *ItemRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	return count, err
}

func (r *ItemRepository) Search(query string) ([]*model.Item, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.Query(`
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items
		WHERE description LIKE ? OR category LIKE ?
		ORDER BY usage_count DESC, description ASC`,
		pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanItems(rows)
}

func (r *ItemRepository) FindByDescription(description string) (*model.Item, error) {
	item := &model.Item{}
	var lastCustID sql.NullString
	err := r.db.QueryRow(`
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items WHERE description = ? COLLATE NOCASE`, description).Scan(
		&item.ID, &item.Description, &item.DefaultPrice, &item.DefaultUnit, &item.DefaultVATRate,
		&item.Category, &item.LastUsedPrice, &lastCustID,
		&item.UsageCount, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastCustID.Valid {
		item.LastCustomerID = lastCustID.String
	}
	return item, nil
}

func (r *ItemRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM items WHERE id = ?", id)
	return err
}

func (r *ItemRepository) GetMostUsed(limit int) ([]*model.Item, error) {
	rows, err := r.db.Query(`
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items
		WHERE usage_count > 0
		ORDER BY usage_count DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanItems(rows)
}

func (r *ItemRepository) GetRecentlyUsed(limit int) ([]*model.Item, error) {
	rows, err := r.db.Query(`
		SELECT id, description, default_price, default_unit, default_vat_rate,
			category, last_used_price, last_customer_id, usage_count, created_at, updated_at
		FROM items
		WHERE usage_count > 0
		ORDER BY updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanItems(rows)
}

func (r *ItemRepository) IncrementUsage(id string, price float64, customerID string) error {
	_, err := r.db.Exec(`
		UPDATE items
		SET usage_count = usage_count + 1,
			last_used_price = ?,
			last_customer_id = ?,
			updated_at = ?
		WHERE id = ?`,
		price, customerID, time.Now(), id)
	return err
}

func (r *ItemRepository) GetExistingCategories() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT category FROM items
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func scanItems(rows *sql.Rows) ([]*model.Item, error) {
	var items []*model.Item
	for rows.Next() {
		item := &model.Item{}
		var lastCustID sql.NullString
		if err := rows.Scan(&item.ID, &item.Description, &item.DefaultPrice,
			&item.DefaultUnit, &item.DefaultVATRate, &item.Category,
			&item.LastUsedPrice, &lastCustID,
			&item.UsageCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if lastCustID.Valid {
			item.LastCustomerID = lastCustID.String
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
