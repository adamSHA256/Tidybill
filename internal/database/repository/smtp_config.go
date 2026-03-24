package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type SmtpConfigRepository struct {
	db *sql.DB
}

func NewSmtpConfigRepository(db *sql.DB) *SmtpConfigRepository {
	return &SmtpConfigRepository{db: db}
}

func (r *SmtpConfigRepository) GetBySupplierID(supplierID string) (*model.SmtpConfig, error) {
	c := &model.SmtpConfig{}
	err := r.db.QueryRow(`
		SELECT id, supplier_id, host, port, username, password_encrypted, from_name, from_email,
			use_starttls, enabled, created_at, updated_at
		FROM smtp_configs WHERE supplier_id = ?`, supplierID).Scan(
		&c.ID, &c.SupplierID, &c.Host, &c.Port, &c.Username, &c.PasswordEncrypted,
		&c.FromName, &c.FromEmail, &c.UseStartTLS, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.HasPassword = c.PasswordEncrypted != ""
	return c, nil
}

func (r *SmtpConfigRepository) List() ([]*model.SmtpConfig, error) {
	rows, err := r.db.Query(`
		SELECT id, supplier_id, host, port, username, password_encrypted, from_name, from_email,
			use_starttls, enabled, created_at, updated_at
		FROM smtp_configs ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.SmtpConfig
	for rows.Next() {
		c := &model.SmtpConfig{}
		if err := rows.Scan(&c.ID, &c.SupplierID, &c.Host, &c.Port, &c.Username, &c.PasswordEncrypted,
			&c.FromName, &c.FromEmail, &c.UseStartTLS, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.HasPassword = c.PasswordEncrypted != ""
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *SmtpConfigRepository) Upsert(config *model.SmtpConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO smtp_configs (id, supplier_id, host, port, username, password_encrypted,
			from_name, from_email, use_starttls, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		config.ID, config.SupplierID, config.Host, config.Port, config.Username, config.PasswordEncrypted,
		config.FromName, config.FromEmail, config.UseStartTLS, config.Enabled, config.CreatedAt, config.UpdatedAt)
	return err
}

func (r *SmtpConfigRepository) Delete(supplierID string) error {
	_, err := r.db.Exec("DELETE FROM smtp_configs WHERE supplier_id = ?", supplierID)
	return err
}

func (r *SmtpConfigRepository) UpdatePassword(supplierID string, encryptedPassword string) error {
	_, err := r.db.Exec("UPDATE smtp_configs SET password_encrypted = ?, updated_at = ? WHERE supplier_id = ?",
		encryptedPassword, time.Now(), supplierID)
	return err
}
