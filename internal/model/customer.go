package model

import "time"

type Customer struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Street         string    `json:"street"`
	City           string    `json:"city"`
	ZIP            string    `json:"zip"`
	Region         string    `json:"region"`
	Country        string    `json:"country"`
	ICO            string    `json:"ico"`
	DIC            string    `json:"dic"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	DefaultVATRate float64   `json:"default_vat_rate"`
	DefaultDueDays int       `json:"default_due_days"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewCustomer() *Customer {
	return &Customer{
		Country:        "CZ",
		DefaultVATRate: 0,
		DefaultDueDays: 14,
	}
}
