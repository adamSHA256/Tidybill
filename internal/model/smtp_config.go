package model

import "time"

type SmtpConfig struct {
	ID                string    `json:"id"`
	SupplierID        string    `json:"supplier_id"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	Username          string    `json:"username"`
	PasswordEncrypted string    `json:"-"`
	HasPassword       bool      `json:"has_password"`
	FromName          string    `json:"from_name"`
	FromEmail         string    `json:"from_email"`
	UseStartTLS       bool      `json:"use_starttls"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewSmtpConfig(supplierID string) *SmtpConfig {
	return &SmtpConfig{
		SupplierID:  supplierID,
		Port:        587,
		UseStartTLS: true,
		Enabled:     false,
	}
}
