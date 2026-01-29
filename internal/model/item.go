package model

import "time"

type Item struct {
	ID             string    `json:"id"`
	Description    string    `json:"description"`
	DefaultPrice   float64   `json:"default_price"`
	DefaultUnit    string    `json:"default_unit"`
	DefaultVATRate float64   `json:"default_vat_rate"`
	Category       string    `json:"category"`
	LastUsedPrice  float64   `json:"last_used_price"`
	LastCustomerID string    `json:"last_customer_id"`
	UsageCount     int       `json:"usage_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewItem() *Item {
	return &Item{
		DefaultUnit:    "ks",
		DefaultVATRate: 0,
	}
}

type CustomerItem struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	ItemID       string    `json:"item_id"`
	LastPrice    float64   `json:"last_price"`
	LastQuantity float64   `json:"last_quantity"`
	UsageCount   int       `json:"usage_count"`
	LastUsedAt   time.Time `json:"last_used_at"`
}
