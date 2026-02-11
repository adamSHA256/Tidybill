package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type InvoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

func (r *InvoiceRepository) Create(inv *model.Invoice) error {
	if inv.ID == "" {
		inv.ID = uuid.New().String()
	}
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO invoices (id, invoice_number, supplier_id, customer_id, bank_account_id, status,
			issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol, currency,
			exchange_rate, subtotal, vat_total, total, notes, internal_notes, language, pdf_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.InvoiceNumber, inv.SupplierID, inv.CustomerID, inv.BankAccountID, inv.Status,
		inv.IssueDate, inv.DueDate, inv.PaidDate, inv.TaxableDate, inv.PaymentMethod, inv.VariableSymbol,
		inv.Currency, inv.ExchangeRate, inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes,
		inv.Language, inv.PDFPath, inv.CreatedAt, inv.UpdatedAt)
	return err
}

func (r *InvoiceRepository) Update(inv *model.Invoice) error {
	inv.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE invoices SET invoice_number=?, customer_id=?, bank_account_id=?, status=?,
			issue_date=?, due_date=?, paid_date=?, taxable_date=?, payment_method=?, variable_symbol=?,
			currency=?, exchange_rate=?, subtotal=?, vat_total=?, total=?, notes=?, internal_notes=?,
			language=?, pdf_path=?, updated_at=?
		WHERE id=?`,
		inv.InvoiceNumber, inv.CustomerID, inv.BankAccountID, inv.Status, inv.IssueDate, inv.DueDate,
		inv.PaidDate, inv.TaxableDate, inv.PaymentMethod, inv.VariableSymbol, inv.Currency, inv.ExchangeRate,
		inv.Subtotal, inv.VATTotal, inv.Total, inv.Notes, inv.InternalNotes, inv.Language, inv.PDFPath,
		inv.UpdatedAt, inv.ID)
	return err
}

func (r *InvoiceRepository) GetByID(id string) (*model.Invoice, error) {
	inv := &model.Invoice{}
	var paidDate sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, invoice_number, supplier_id, customer_id, bank_account_id, status,
			issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
			currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
			language, pdf_path, created_at, updated_at
		FROM invoices WHERE id = ?`, id).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.SupplierID, &inv.CustomerID, &inv.BankAccountID, &inv.Status,
		&inv.IssueDate, &inv.DueDate, &paidDate, &inv.TaxableDate, &inv.PaymentMethod, &inv.VariableSymbol,
		&inv.Currency, &inv.ExchangeRate, &inv.Subtotal, &inv.VATTotal, &inv.Total, &inv.Notes, &inv.InternalNotes,
		&inv.Language, &inv.PDFPath, &inv.CreatedAt, &inv.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if paidDate.Valid {
		inv.PaidDate = &paidDate.Time
	}
	return inv, err
}

func (r *InvoiceRepository) List(status model.InvoiceStatus, customerID string) ([]*model.Invoice, error) {
	query := `SELECT id, invoice_number, supplier_id, customer_id, bank_account_id, status,
		issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
		currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
		language, pdf_path, created_at, updated_at FROM invoices WHERE 1=1`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if customerID != "" {
		query += " AND customer_id = ?"
		args = append(args, customerID)
	}
	query += " ORDER BY issue_date DESC, invoice_number DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*model.Invoice
	for rows.Next() {
		inv := &model.Invoice{}
		var paidDate sql.NullTime
		if err := rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.SupplierID, &inv.CustomerID, &inv.BankAccountID,
			&inv.Status, &inv.IssueDate, &inv.DueDate, &paidDate, &inv.TaxableDate, &inv.PaymentMethod,
			&inv.VariableSymbol, &inv.Currency, &inv.ExchangeRate, &inv.Subtotal, &inv.VATTotal, &inv.Total,
			&inv.Notes, &inv.InternalNotes, &inv.Language, &inv.PDFPath, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		if paidDate.Valid {
			inv.PaidDate = &paidDate.Time
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) ListUnpaid() ([]*model.Invoice, error) {
	rows, err := r.db.Query(`
		SELECT id, invoice_number, supplier_id, customer_id, bank_account_id, status,
			issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
			currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes,
			language, pdf_path, created_at, updated_at
		FROM invoices WHERE status NOT IN ('paid', 'cancelled')
		ORDER BY due_date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*model.Invoice
	for rows.Next() {
		inv := &model.Invoice{}
		var paidDate sql.NullTime
		if err := rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.SupplierID, &inv.CustomerID, &inv.BankAccountID,
			&inv.Status, &inv.IssueDate, &inv.DueDate, &paidDate, &inv.TaxableDate, &inv.PaymentMethod,
			&inv.VariableSymbol, &inv.Currency, &inv.ExchangeRate, &inv.Subtotal, &inv.VATTotal, &inv.Total,
			&inv.Notes, &inv.InternalNotes, &inv.Language, &inv.PDFPath, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		if paidDate.Valid {
			inv.PaidDate = &paidDate.Time
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) UpdateStatus(id string, status model.InvoiceStatus) error {
	_, err := r.db.Exec("UPDATE invoices SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id)
	return err
}

func (r *InvoiceRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM invoices WHERE id = ?", id)
	return err
}

func (r *InvoiceRepository) GetNextNumber(supplierID string, prefix string) (string, error) {
	year := time.Now().Format("06")
	pattern := prefix + year + "-%"

	var maxNum int
	err := r.db.QueryRow(`
		SELECT COALESCE(MAX(CAST(SUBSTR(invoice_number, -5) AS INTEGER)), 0)
		FROM invoices WHERE supplier_id = ? AND invoice_number LIKE ?`, supplierID, pattern).Scan(&maxNum)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%s-%05d", prefix, year, maxNum+1), nil
}

func GenerateVariableSymbol(invoiceNumber string) string {
	vs := strings.ReplaceAll(invoiceNumber, "-", "")
	vs = strings.TrimLeft(vs, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	return vs
}

func (r *InvoiceRepository) CountUnpaid() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM invoices WHERE status NOT IN ('paid', 'cancelled')").Scan(&count)
	return count, err
}

func (r *InvoiceRepository) CountOverdue() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE status NOT IN ('paid', 'cancelled') AND due_date < DATE('now')`).Scan(&count)
	return count, err
}
