package model

import "time"

type BankAccount struct {
	ID            string    `json:"id"`
	SupplierID    string    `json:"supplier_id"`
	Name          string    `json:"name"`
	AccountNumber string    `json:"account_number"`
	IBAN          string    `json:"iban"`
	SWIFT         string    `json:"swift"`
	Currency      string    `json:"currency"`
	IsDefault     bool      `json:"is_default"`
	QRType        string    `json:"qr_type"` // "spayd", "pay_by_square", "epc", "none"
	CreatedAt     time.Time `json:"created_at"`
}

func NewBankAccount(supplierID string) *BankAccount {
	return &BankAccount{
		SupplierID: supplierID,
		Currency:   "CZK",
		IsDefault:  false,
		QRType:     "spayd",
	}
}
