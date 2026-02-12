package model

import (
	"math"
	"strings"
	"time"
)

const (
	MaxDescriptionLen = 100
	MaxCategoryLen    = 50
)

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

	// Joined fields (populated by queries with JOIN, not stored in DB)
	ItemDescription string  `json:"item_description,omitempty"`
	ItemCategory    string  `json:"item_category,omitempty"`
	ItemDefaultUnit string  `json:"item_default_unit,omitempty"`
	ItemDefaultVAT  float64 `json:"item_default_vat,omitempty"`
}

func RoundMoney(amount float64) float64 {
	return math.Round(amount*100) / 100
}

func NormalizeCategory(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}
