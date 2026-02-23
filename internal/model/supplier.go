package model

import (
	"time"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type Supplier struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Street        string    `json:"street"`
	City          string    `json:"city"`
	ZIP           string    `json:"zip"`
	Country       string    `json:"country"`
	ICO           string    `json:"ico"`
	DIC           string    `json:"dic"`
	ICDPH         string    `json:"ic_dph"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	LogoPath      string    `json:"logo_path"`
	IsVATPayer    bool      `json:"is_vat_payer"`
	IsDefault     bool      `json:"is_default"`
	InvoicePrefix string    `json:"invoice_prefix"`
	Website       string 	`json:"website"`
	Notes         string    `json:"notes"`
	Language      string    `json:"language"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewSupplier() *Supplier {
	return &Supplier{
		Country:       "CZ",
		InvoicePrefix: "VF",
		Language:      string(i18n.GetLang()),
		IsVATPayer:    false,
		IsDefault:     true,
		Website:       "printmoney.usd",
	}
}
