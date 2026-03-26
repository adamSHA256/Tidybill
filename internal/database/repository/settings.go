package repository

import "database/sql"

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (r *SettingsRepository) Set(key, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value)
	return err
}

// SetDefault sets a key only if it doesn't already exist in the database
func (r *SettingsRepository) SetDefault(key, value string) error {
	_, err := r.db.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)",
		key, value)
	return err
}
