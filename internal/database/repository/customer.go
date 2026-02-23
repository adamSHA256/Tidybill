package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(c *model.Customer) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO customers (id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
		c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CustomerRepository) Update(c *model.Customer) error {
	c.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE customers SET name=?, street=?, city=?, zip=?, region=?, country=?, ico=?, dic=?, ic_dph=?,
			email=?, phone=?, default_vat_rate=?, default_due_days=?, notes=?, updated_at=?
		WHERE id=?`,
		c.Name, c.Street, c.City, c.ZIP, c.Region, c.Country, c.ICO, c.DIC, c.ICDPH,
		c.Email, c.Phone, c.DefaultVATRate, c.DefaultDueDays, c.Notes, c.UpdatedAt, c.ID)
	return err
}

func (r *CustomerRepository) GetByID(id string) (*model.Customer, error) {
	c := &model.Customer{}
	err := r.db.QueryRow(`
		SELECT id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes, created_at, updated_at
		FROM customers WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Street, &c.City, &c.ZIP, &c.Region, &c.Country, &c.ICO, &c.DIC, &c.ICDPH,
		&c.Email, &c.Phone, &c.DefaultVATRate, &c.DefaultDueDays, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *CustomerRepository) List() ([]*model.Customer, error) {
	rows, err := r.db.Query(`
		SELECT id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes, created_at, updated_at
		FROM customers ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*model.Customer
	for rows.Next() {
		c := &model.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Street, &c.City, &c.ZIP, &c.Region, &c.Country,
			&c.ICO, &c.DIC, &c.ICDPH, &c.Email, &c.Phone, &c.DefaultVATRate, &c.DefaultDueDays,
			&c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

func (r *CustomerRepository) Search(query string) ([]*model.Customer, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.Query(`
		SELECT id, name, street, city, zip, region, country, ico, dic, ic_dph,
			email, phone, default_vat_rate, default_due_days, notes, created_at, updated_at
		FROM customers WHERE name LIKE ? OR ico LIKE ? ORDER BY name ASC`, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*model.Customer
	for rows.Next() {
		c := &model.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Street, &c.City, &c.ZIP, &c.Region, &c.Country,
			&c.ICO, &c.DIC, &c.ICDPH, &c.Email, &c.Phone, &c.DefaultVATRate, &c.DefaultDueDays,
			&c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

func (r *CustomerRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM customers WHERE id = ?", id)
	return err
}

func (r *CustomerRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM customers").Scan(&count)
	return count, err
}
