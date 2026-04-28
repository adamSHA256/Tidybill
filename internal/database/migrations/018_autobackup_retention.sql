-- Auto-backup retention (Phase E).
-- After each successful auto-backup upload, the prune step deletes obsolete
-- cloud blobs per a GFS (grandfather-father-son) schedule.
--
-- Defaults:
--   retention_enabled            = 1   -- prune after every backup
--   retention_keep_recent_days   = 7   -- keep ALL backups within last N days
--   retention_keep_daily_days    = 30  -- after N days, keep only most recent per UTC day
--   retention_keep_weekly_months = 6   -- after that, keep most recent per ISO week
--   retention_keep_monthly_months = 0  -- 0 = keep monthly forever; >0 = delete monthly older than this many months

INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.retention_enabled', '1');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.retention_keep_recent_days', '7');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.retention_keep_daily_days', '30');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.retention_keep_weekly_months', '6');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cloud.autobackup.retention_keep_monthly_months', '0');
