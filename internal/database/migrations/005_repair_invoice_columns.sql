-- Repair data corruption from broken 004 migration.
-- The broken 004 used "INSERT INTO ... SELECT * FROM" which copies by column
-- position. Since template_id was added via ALTER TABLE (last position) but
-- placed before created_at/updated_at in the new table, the three columns
-- got shifted:
--   template_id  <- actually holds old created_at
--   created_at   <- actually holds old updated_at
--   updated_at   <- actually holds old template_id (e.g. "default")
--
-- This migration detects corruption by checking if updated_at contains a
-- non-datetime string, and swaps the columns back. On clean databases
-- (fixed 004 or fresh installs), it's a no-op table recreation.

PRAGMA foreign_keys = OFF;

CREATE TABLE invoices_repair (
    id TEXT PRIMARY KEY,
    invoice_number TEXT NOT NULL,
    supplier_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    bank_account_id TEXT NOT NULL,
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
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (bank_account_id) REFERENCES bank_accounts(id)
);

-- Detect corruption: if updated_at doesn't look like a datetime (YYYY-MM-DD...),
-- swap the three columns back. Otherwise keep as-is.
INSERT INTO invoices_repair (id, invoice_number, supplier_id, customer_id, bank_account_id,
    status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
    currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes, language,
    pdf_path, template_id, created_at, updated_at)
SELECT id, invoice_number, supplier_id, customer_id, bank_account_id,
    status, issue_date, due_date, paid_date, taxable_date, payment_method, variable_symbol,
    currency, exchange_rate, subtotal, vat_total, total, notes, internal_notes, language,
    pdf_path,
    CASE WHEN updated_at NOT LIKE '____-__-__%' THEN updated_at ELSE template_id END,
    CASE WHEN updated_at NOT LIKE '____-__-__%' THEN template_id ELSE created_at END,
    CASE WHEN updated_at NOT LIKE '____-__-__%' THEN created_at ELSE updated_at END
FROM invoices;

DROP TABLE invoices;
ALTER TABLE invoices_repair RENAME TO invoices;

CREATE UNIQUE INDEX idx_invoices_supplier_number ON invoices(supplier_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_customer ON invoices(customer_id);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);

PRAGMA foreign_keys = ON;
