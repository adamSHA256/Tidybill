-- Initial schema

CREATE TABLE suppliers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    street TEXT,
    city TEXT,
    zip TEXT,
    country TEXT DEFAULT 'CZ',
    ico TEXT,
    dic TEXT,
    phone TEXT,
    email TEXT,
    logo_path TEXT,
    is_vat_payer INTEGER DEFAULT 0,
    is_default INTEGER DEFAULT 0,
    invoice_prefix TEXT DEFAULT 'VF',
    notes TEXT,
    language TEXT DEFAULT 'cs',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bank_accounts (
    id TEXT PRIMARY KEY,
    supplier_id TEXT NOT NULL,
    name TEXT,
    account_number TEXT,
    iban TEXT,
    swift TEXT,
    currency TEXT DEFAULT 'CZK',
    is_default INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE CASCADE
);

CREATE INDEX idx_bank_accounts_supplier ON bank_accounts(supplier_id);

CREATE TABLE customers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    street TEXT,
    city TEXT,
    zip TEXT,
    region TEXT,
    country TEXT DEFAULT 'CZ',
    ico TEXT,
    dic TEXT,
    email TEXT,
    phone TEXT,
    default_vat_rate REAL DEFAULT 0,
    default_due_days INTEGER DEFAULT 14,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_customers_name ON customers(name);
CREATE INDEX idx_customers_ico ON customers(ico);

CREATE TABLE invoices (
    id TEXT PRIMARY KEY,
    invoice_number TEXT UNIQUE NOT NULL,
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
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (bank_account_id) REFERENCES bank_accounts(id)
);

CREATE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_customer ON invoices(customer_id);
CREATE INDEX idx_invoices_issue_date ON invoices(issue_date);

CREATE TABLE invoice_items (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    item_id TEXT,
    description TEXT NOT NULL,
    quantity REAL DEFAULT 1,
    unit TEXT DEFAULT 'ks',
    unit_price REAL DEFAULT 0,
    vat_rate REAL DEFAULT 0,
    subtotal REAL DEFAULT 0,
    vat_amount REAL DEFAULT 0,
    total REAL DEFAULT 0,
    position INTEGER DEFAULT 0,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE SET NULL
);

CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);

CREATE TABLE items (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    default_price REAL DEFAULT 0,
    default_unit TEXT DEFAULT 'ks',
    default_vat_rate REAL DEFAULT 0,
    category TEXT,
    last_used_price REAL DEFAULT 0,
    last_customer_id TEXT,
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE customer_items (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    last_price REAL DEFAULT 0,
    last_quantity REAL DEFAULT 1,
    usage_count INTEGER DEFAULT 0,
    last_used_at DATETIME,
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
    UNIQUE(customer_id, item_id)
);

CREATE TABLE vat_rates (
    id TEXT PRIMARY KEY,
    rate REAL NOT NULL,
    name TEXT,
    is_default INTEGER DEFAULT 0,
    country TEXT DEFAULT 'CZ'
);

INSERT INTO vat_rates (id, rate, name, is_default, country) VALUES
    ('cz-0', 0, 'Bez DPH', 1, 'CZ'),
    ('cz-12', 12, 'Snížená 12%', 0, 'CZ'),
    ('cz-21', 21, 'Základní 21%', 0, 'CZ'),
    ('sk-0', 0, 'Bez DPH', 1, 'SK'),
    ('sk-10', 10, 'Znížená 10%', 0, 'SK'),
    ('sk-20', 20, 'Základná 20%', 0, 'SK');

CREATE TABLE pdf_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_code TEXT NOT NULL,
    config_json TEXT,
    is_default INTEGER DEFAULT 0,
    supplier_id TEXT,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE CASCADE
);

INSERT INTO pdf_templates (id, name, template_code, is_default) VALUES
    ('default', 'Výchozí šablona', 'default', 1);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
);
