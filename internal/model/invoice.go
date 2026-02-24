package model

import (
	"time"

	"github.com/adamSHA256/tidybill/internal/i18n"
)

type InvoiceStatus string

const (
	StatusDraft         InvoiceStatus = "draft"
	StatusCreated       InvoiceStatus = "created"
	StatusSent          InvoiceStatus = "sent"
	StatusPaid          InvoiceStatus = "paid"
	StatusOverdue       InvoiceStatus = "overdue"
	StatusPartiallyPaid InvoiceStatus = "partially_paid"
	StatusCancelled     InvoiceStatus = "cancelled"
)

type Invoice struct {
	ID             string        `json:"id"`
	InvoiceNumber  string        `json:"invoice_number"`
	SupplierID     string        `json:"supplier_id"`
	CustomerID     string        `json:"customer_id"`
	BankAccountID  string        `json:"bank_account_id"`
	Status         InvoiceStatus `json:"status"`
	IssueDate      time.Time     `json:"issue_date"`
	DueDate        time.Time     `json:"due_date"`
	PaidDate       *time.Time    `json:"paid_date"`
	TaxableDate    time.Time     `json:"taxable_date"`
	PaymentMethod  string        `json:"payment_method"`
	VariableSymbol string        `json:"variable_symbol"`
	Currency       string        `json:"currency"`
	ExchangeRate   float64       `json:"exchange_rate"`
	Subtotal       float64       `json:"subtotal"`
	VATTotal       float64       `json:"vat_total"`
	Total          float64       `json:"total"`
	Notes          string        `json:"notes"`
	InternalNotes  string        `json:"internal_notes"`
	Language       string        `json:"language"`
	PDFPath        string        `json:"pdf_path"`
	TemplateID     string        `json:"template_id"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`

	// Loaded relations (not stored in DB)
	Items    []InvoiceItem `json:"items,omitempty"`
	Customer *Customer     `json:"customer,omitempty"`
	Supplier *Supplier     `json:"supplier,omitempty"`
}

func NewInvoice(supplierID, customerID, bankAccountID string) *Invoice {
	now := time.Now()
	return &Invoice{
		SupplierID:    supplierID,
		CustomerID:    customerID,
		BankAccountID: bankAccountID,
		Status:        StatusCreated,
		IssueDate:     now,
		DueDate:       now.AddDate(0, 0, 14),
		TaxableDate:   now,
		PaymentMethod: "bank_transfer",
		Currency:      "CZK",
		ExchangeRate:  1.0,
		Language:      string(i18n.GetLang()),
	}
}

type InvoiceItem struct {
	ID          string  `json:"id"`
	InvoiceID   string  `json:"invoice_id"`
	ItemID      string  `json:"item_id"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	VATRate     float64 `json:"vat_rate"`
	Subtotal    float64 `json:"subtotal"`
	VATAmount   float64 `json:"vat_amount"`
	Total       float64 `json:"total"`
	Position    int     `json:"position"`
}

func NewInvoiceItem(invoiceID string) *InvoiceItem {
	return &InvoiceItem{
		InvoiceID: invoiceID,
		Quantity:  1,
		Unit:      "ks",
		VATRate:   0,
	}
}

func (i *InvoiceItem) Calculate() {
	i.Subtotal = RoundMoney(i.Quantity * i.UnitPrice)
	i.VATAmount = RoundMoney(i.Subtotal * i.VATRate / 100)
	i.Total = RoundMoney(i.Subtotal + i.VATAmount)
}

func (i *Invoice) IsOverdue() bool {
	if (i.Status == StatusDraft || i.Status == StatusCreated || 
	   i.Status == StatusCancelled) || !i.DueDate.Before(time.Now()) {
		return false
	}
	return true
}
