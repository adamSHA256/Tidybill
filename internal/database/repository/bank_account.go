package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/adamSHA256/tidybill/internal/model"
)

type BankAccountRepository struct {
	db DBTX
}

func NewBankAccountRepository(db DBTX) *BankAccountRepository {
	return &BankAccountRepository{db: db}
}

func (r *BankAccountRepository) WithDB(db DBTX) *BankAccountRepository {
	return &BankAccountRepository{db: db}
}

func (r *BankAccountRepository) Create(ba *model.BankAccount) error {
	if ba.ID == "" {
		ba.ID = uuid.New().String()
	}
	ba.CreatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO bank_accounts (id, supplier_id, name, account_number, iban, swift, currency, is_default, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ba.ID, ba.SupplierID, ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT, ba.Currency, ba.IsDefault, ba.CreatedAt)
	return err
}

func (r *BankAccountRepository) Update(ba *model.BankAccount) error {
	_, err := r.db.Exec(`
		UPDATE bank_accounts SET name=?, account_number=?, iban=?, swift=?, currency=?, is_default=?
		WHERE id=?`,
		ba.Name, ba.AccountNumber, ba.IBAN, ba.SWIFT, ba.Currency, ba.IsDefault, ba.ID)
	return err
}

func (r *BankAccountRepository) GetByID(id string) (*model.BankAccount, error) {
	ba := &model.BankAccount{}
	err := r.db.QueryRow(`
		SELECT id, supplier_id, name, account_number, iban, swift, currency, is_default, created_at
		FROM bank_accounts WHERE id = ?`, id).Scan(
		&ba.ID, &ba.SupplierID, &ba.Name, &ba.AccountNumber, &ba.IBAN, &ba.SWIFT, &ba.Currency, &ba.IsDefault, &ba.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ba, err
}

func (r *BankAccountRepository) GetBySupplier(supplierID string) ([]*model.BankAccount, error) {
	rows, err := r.db.Query(`
		SELECT id, supplier_id, name, account_number, iban, swift, currency, is_default, created_at
		FROM bank_accounts WHERE supplier_id = ? ORDER BY is_default DESC, currency ASC`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.BankAccount
	for rows.Next() {
		ba := &model.BankAccount{}
		if err := rows.Scan(&ba.ID, &ba.SupplierID, &ba.Name, &ba.AccountNumber, &ba.IBAN,
			&ba.SWIFT, &ba.Currency, &ba.IsDefault, &ba.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, ba)
	}
	return accounts, rows.Err()
}

func (r *BankAccountRepository) GetDefaultForSupplier(supplierID string) (*model.BankAccount, error) {
	ba := &model.BankAccount{}
	err := r.db.QueryRow(`
		SELECT id, supplier_id, name, account_number, iban, swift, currency, is_default, created_at
		FROM bank_accounts WHERE supplier_id = ? AND is_default = 1 LIMIT 1`, supplierID).Scan(
		&ba.ID, &ba.SupplierID, &ba.Name, &ba.AccountNumber, &ba.IBAN, &ba.SWIFT, &ba.Currency, &ba.IsDefault, &ba.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ba, err
}

func (r *BankAccountRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM bank_accounts WHERE id = ?", id)
	return err
}

func (r *BankAccountRepository) ClearDefaultsForCurrency(supplierID, currency string) error {
	_, err := r.db.Exec("UPDATE bank_accounts SET is_default = 0 WHERE supplier_id = ? AND currency = ?", supplierID, currency)
	return err
}

func (r *BankAccountRepository) CountBySupplier(supplierID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM bank_accounts WHERE supplier_id = ?", supplierID).Scan(&count)
	return count, err
}
