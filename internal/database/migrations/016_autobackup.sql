-- Auto-backup settings and write-change detection.
-- Uses IF NOT EXISTS / INSERT OR IGNORE so re-running is safe.

-- Auto-backup settings
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.enabled', '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.transport_id', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.idle_minutes', '5');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.last_run_at', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.last_error', '');

-- Write-change detection: counter bumped by triggers below, timestamp updated too.
INSERT OR IGNORE INTO settings (key, value) VALUES ('data.write_counter', '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('data.last_write_at', '');

-- Triggers to detect data changes. We watch the main user-data tables;
-- changes to settings/cloud_configs/schema_migrations are not user-data writes.

CREATE TRIGGER IF NOT EXISTS ab_bump_invoices_ins AFTER INSERT ON invoices BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_invoices_upd AFTER UPDATE ON invoices BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_invoices_del AFTER DELETE ON invoices BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_invoice_items_ins AFTER INSERT ON invoice_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_invoice_items_upd AFTER UPDATE ON invoice_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_invoice_items_del AFTER DELETE ON invoice_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_customers_ins AFTER INSERT ON customers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_customers_upd AFTER UPDATE ON customers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_customers_del AFTER DELETE ON customers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_suppliers_ins AFTER INSERT ON suppliers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_suppliers_upd AFTER UPDATE ON suppliers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_suppliers_del AFTER DELETE ON suppliers BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_bank_accounts_ins AFTER INSERT ON bank_accounts BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_bank_accounts_upd AFTER UPDATE ON bank_accounts BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_bank_accounts_del AFTER DELETE ON bank_accounts BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_items_ins AFTER INSERT ON items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_items_upd AFTER UPDATE ON items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_items_del AFTER DELETE ON items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_customer_items_ins AFTER INSERT ON customer_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_customer_items_upd AFTER UPDATE ON customer_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_customer_items_del AFTER DELETE ON customer_items BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_pdf_templates_ins AFTER INSERT ON pdf_templates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_pdf_templates_upd AFTER UPDATE ON pdf_templates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_pdf_templates_del AFTER DELETE ON pdf_templates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_vat_rates_ins AFTER INSERT ON vat_rates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_vat_rates_upd AFTER UPDATE ON vat_rates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_vat_rates_del AFTER DELETE ON vat_rates BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;

CREATE TRIGGER IF NOT EXISTS ab_bump_smtp_configs_ins AFTER INSERT ON smtp_configs BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_smtp_configs_upd AFTER UPDATE ON smtp_configs BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
CREATE TRIGGER IF NOT EXISTS ab_bump_smtp_configs_del AFTER DELETE ON smtp_configs BEGIN
  UPDATE settings SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT) WHERE key = 'data.write_counter';
  UPDATE settings SET value = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE key = 'data.last_write_at';
END;
