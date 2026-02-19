-- Make bank_account_id nullable so cash/non-bank invoices don't require a bank account.
-- Remove the FK constraint for bank_account_id (empty string or NULL = no bank account).

PRAGMA foreign_keys = OFF;

CREATE TABLE invoices_new (
    id TEXT PRIMARY KEY,
    invoice_number TEXT NOT NULL,
    supplier_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    bank_account_id TEXT,
    status TEXT DEFAULT 'draft',
    issue_date DATE NOT NULL,
    due_date DATE NOT NULL,
    paid_date DATE,
    taxable_date DATE,
    payment_method TEXT DEFAULT 'bank_transfer',
    variable_symbol TEXT,
    currency TEXT DEFAULT 'CZK',
    exchange_rate REAL DEFAULT 1.0,
    subtotal REAL DEFAULT 0,
    vat_total REAL DEFAULT 0,
    total REAL DEFAULT 0,
    notes TEXT,
    internal_notes TEXT,
    language TEXT DEFAULT 'cs',
    pdf_path TEXT,
    template_id TEXT DEFAULT 'default',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

INSERT INTO invoices_new (id, invoice_number, supplier_id, customer_id, bank_account_id,
    status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
    currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes, language,
    pdf_path, template_id, created_at, updated_at)
SELECT id, invoice_number, supplier_id, customer_id, bank_account_id,
    status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
    currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes, language,
    pdf_path, template_id, created_at, updated_at
FROM invoices;

DROP TABLE invoices;
ALTER TABLE invoices_new RENAME TO invoices;

CREATE UNIQUE INDEX idx_invoices_supplier_number ON invoices(supplier_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_customer ON invoices(customer_id);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);

PRAGMA foreign_keys = ON;
