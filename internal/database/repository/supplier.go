package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/user/invoice-app/internal/model"
)

type SupplierRepository struct {
	db *sql.DB
}

func NewSupplierRepository(db *sql.DB) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(s *model.Supplier) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO suppliers (id, name, street, city, zip, country, ico, dic, phone, email, website,
			logo_path, is_vat_payer, is_default, invoice_prefix, notes, language, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.Phone, s.Email, s.Website,
		s.LogoPath, s.IsVATPayer, s.IsDefault, s.InvoicePrefix, s.Notes, s.Language, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *SupplierRepository) Update(s *model.Supplier) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE suppliers SET name=?, street=?, city=?, zip=?, country=?, ico=?, dic=?, phone=?, email=?, website=?,
			logo_path=?, is_vat_payer=?, is_default=?, invoice_prefix=?, notes=?, language=?, updated_at=?
		WHERE id=?`,
		s.Name, s.Street, s.City, s.ZIP, s.Country, s.ICO, s.DIC, s.Phone, s.Email, s.Website,
		s.LogoPath, s.IsVATPayer, s.IsDefault, s.InvoicePrefix, s.Notes, s.Language, s.UpdatedAt, s.ID)
	return err
}

func (r *SupplierRepository) GetByID(id string) (*model.Supplier, error) {
	s := &model.Supplier{}
	err := r.db.QueryRow(`
		SELECT id, name, street, city, zip, country, ico, dic, phone, email, website,
			logo_path, is_vat_payer, is_default, invoice_prefix, notes, language, created_at, updated_at
		FROM suppliers WHERE id = ?`, id).Scan(
		&s.ID, &s.Name, &s.Street, &s.City, &s.ZIP, &s.Country, &s.ICO, &s.DIC, &s.Phone, &s.Email, &s.Website,
		&s.LogoPath, &s.IsVATPayer, &s.IsDefault, &s.InvoicePrefix, &s.Notes, &s.Language, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *SupplierRepository) GetDefault() (*model.Supplier, error) {
	s := &model.Supplier{}
	err := r.db.QueryRow(`
		SELECT id, name, street, city, zip, country, ico, dic, phone, email, website,
			logo_path, is_vat_payer, is_default, invoice_prefix, notes, language, created_at, updated_at
		FROM suppliers WHERE is_default = 1 LIMIT 1`).Scan(
		&s.ID, &s.Name, &s.Street, &s.City, &s.ZIP, &s.Country, &s.ICO, &s.DIC, &s.Phone, &s.Email, &s.Website,
		&s.LogoPath, &s.IsVATPayer, &s.IsDefault, &s.InvoicePrefix, &s.Notes, &s.Language, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *SupplierRepository) List() ([]*model.Supplier, error) {
	rows, err := r.db.Query(`
		SELECT id, name, street, city, zip, country, ico, dic, phone, email, website,
			logo_path, is_vat_payer, is_default, invoice_prefix, notes, language, created_at, updated_at
		FROM suppliers ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []*model.Supplier
	for rows.Next() {
		s := &model.Supplier{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Street, &s.City, &s.ZIP, &s.Country, &s.ICO, &s.DIC,
			&s.Phone, &s.Email, &s.Website, &s.LogoPath, &s.IsVATPayer, &s.IsDefault, &s.InvoicePrefix,
			&s.Notes, &s.Language, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, rows.Err()
}

func (r *SupplierRepository) SetDefault(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE suppliers SET is_default = 0"); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE suppliers SET is_default = 1 WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SupplierRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM suppliers WHERE id = ?", id)
	return err
}

func (r *SupplierRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM suppliers").Scan(&count)
	return count, err
}
