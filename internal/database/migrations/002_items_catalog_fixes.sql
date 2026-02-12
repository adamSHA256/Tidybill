-- Migration 002: Items catalog fixes
-- Fixes: C3 (FK on last_customer_id), C4 (table ordering),
--         S6 (search index), S7 (COLLATE NOCASE), M5 (last_used_at DEFAULT)

DROP TABLE IF EXISTS customer_items;
DROP TABLE IF EXISTS items;

CREATE TABLE items (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL COLLATE NOCASE,
    default_price REAL DEFAULT 0,
    default_unit TEXT DEFAULT 'ks',
    default_vat_rate REAL DEFAULT 0,
    category TEXT COLLATE NOCASE,
    last_used_price REAL DEFAULT 0,
    last_customer_id TEXT,
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (last_customer_id) REFERENCES customers(id) ON DELETE SET NULL
);

CREATE INDEX idx_items_description ON items(description);
CREATE INDEX idx_items_category ON items(category);
CREATE INDEX idx_items_usage_count ON items(usage_count);

CREATE TABLE customer_items (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    last_price REAL DEFAULT 0,
    last_quantity REAL DEFAULT 1,
    usage_count INTEGER DEFAULT 0,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
    UNIQUE(customer_id, item_id)
);

CREATE INDEX idx_customer_items_customer ON customer_items(customer_id);
CREATE INDEX idx_customer_items_item ON customer_items(item_id);
