-- Auto-sync settings (Phase D).
-- The sync check is opt-in (default off). When enabled, the AutoSyncService
-- compares the most recent cloud backup timestamp against last_synced_at and
-- last_pulled_at to decide whether to prompt the user for an import.

INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.enabled', '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.interval_minutes', '60');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.check_on_start', '1');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.last_check_at', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.last_pulled_at', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.last_synced_at', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.last_error', '');
-- last_skipped_provider_id remembers a cloud blob the user explicitly skipped
-- so we don't re-prompt for the same one. Cleared whenever a newer blob shows up.
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autosync.last_skipped_provider_id', '');
