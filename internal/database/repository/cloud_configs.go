package repository

import (
	"context"
	"database/sql"
)

type CloudConfig struct {
	TransportID  string
	Enabled      bool
	AccountLabel sql.NullString
	PublicConfig sql.NullString // JSON
}

type CloudConfigsRepository struct {
	db *sql.DB
}

func NewCloudConfigsRepository(db *sql.DB) *CloudConfigsRepository {
	return &CloudConfigsRepository{db: db}
}

func (r *CloudConfigsRepository) ListEnabled(ctx context.Context, prefix string) ([]CloudConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT transport_id, enabled, account_label, public_config
         FROM cloud_configs
         WHERE enabled = 1 AND transport_id LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CloudConfig
	for rows.Next() {
		var c CloudConfig
		var en int
		if err := rows.Scan(&c.TransportID, &en, &c.AccountLabel, &c.PublicConfig); err != nil {
			return nil, err
		}
		c.Enabled = en != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CloudConfigsRepository) Upsert(ctx context.Context, c CloudConfig) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO cloud_configs (transport_id, enabled, account_label, public_config, created_at, updated_at)
        VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
        ON CONFLICT(transport_id) DO UPDATE SET
            enabled = excluded.enabled,
            account_label = excluded.account_label,
            public_config = excluded.public_config,
            updated_at = datetime('now')`,
		c.TransportID,
		boolToInt(c.Enabled),
		nullSQLString(c.AccountLabel),
		nullSQLString(c.PublicConfig),
	)
	return err
}

func (r *CloudConfigsRepository) Disable(ctx context.Context, transportID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cloud_configs SET enabled = 0, updated_at = datetime('now') WHERE transport_id = ?`,
		transportID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullSQLString(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// InsertUpload records a completed cloud upload.
func (r *CloudConfigsRepository) InsertUpload(ctx context.Context, transportID, filename, providerID string, sizeBytes int, encrypted bool) error {
	enc := 0
	if encrypted {
		enc = 1
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cloud_uploads (transport_id, filename, size_bytes, encrypted, provider_id)
         VALUES (?, ?, ?, ?, ?)`,
		transportID, filename, sizeBytes, enc, providerID)
	return err
}
