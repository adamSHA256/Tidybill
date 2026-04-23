-- Cloud transport configuration (public, non-secret).
-- Secrets live in the OS keychain, not in this table.
--
-- Uses IF NOT EXISTS everywhere because the migration runner at
-- internal/database/database.go does NOT wrap a migration in a
-- transaction. If a later statement fails after an earlier statement
-- succeeded, re-running the migration must not error on the object
-- that already exists.

CREATE TABLE IF NOT EXISTS cloud_configs (
    transport_id  TEXT PRIMARY KEY,
    enabled       INTEGER NOT NULL DEFAULT 0,
    account_label TEXT,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    public_config TEXT
);

CREATE TABLE IF NOT EXISTS cloud_uploads (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    transport_id TEXT NOT NULL,
    filename     TEXT NOT NULL,
    size_bytes   INTEGER NOT NULL,
    encrypted    INTEGER NOT NULL DEFAULT 1,
    provider_id  TEXT,
    uploaded_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    note         TEXT
);

CREATE INDEX IF NOT EXISTS idx_cloud_uploads_transport
    ON cloud_uploads(transport_id, uploaded_at DESC);
