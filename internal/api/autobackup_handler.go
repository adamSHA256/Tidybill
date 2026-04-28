package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// GET /api/cloud/autobackup/status
func (s *Server) handleAutoBackupStatus(w http.ResponseWriter, r *http.Request) {
	enabled, _ := s.settings.Get("cloud.autobackup.enabled")
	transportID, _ := s.settings.Get("cloud.autobackup.transport_id")
	idleMinStr, _ := s.settings.Get("cloud.autobackup.idle_minutes")
	lastRunAt, _ := s.settings.Get("cloud.autobackup.last_run_at")
	lastError, _ := s.settings.Get("cloud.autobackup.last_error")
	inProgress, _ := s.settings.Get("cloud.autobackup.in_progress")

	retentionEnabled, _ := s.settings.Get("cloud.autobackup.retention_enabled")
	retentionRecent := readIntSetting(s, "cloud.autobackup.retention_keep_recent_days", 7)
	retentionDaily := readIntSetting(s, "cloud.autobackup.retention_keep_daily_days", 30)
	retentionWeekly := readIntSetting(s, "cloud.autobackup.retention_keep_weekly_months", 6)
	retentionMonthly := readIntSetting(s, "cloud.autobackup.retention_keep_monthly_months", 0)

	idleMin := 5
	if n, err := strconv.Atoi(idleMinStr); err == nil && n > 0 {
		idleMin = n
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":      enabled == "1",
		"transport_id": transportID,
		"idle_minutes": idleMin,
		"last_run_at":  lastRunAt,
		"last_error":   lastError,
		"in_progress":  inProgress == "1",
		"retention": map[string]any{
			"enabled":              retentionEnabled == "1",
			"keep_recent_days":     retentionRecent,
			"keep_daily_days":      retentionDaily,
			"keep_weekly_months":   retentionWeekly,
			"keep_monthly_months":  retentionMonthly,
		},
	})
}

func readIntSetting(s *Server, key string, def int) int {
	v, _ := s.settings.Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// PUT /api/cloud/autobackup/settings
func (s *Server) handleAutoBackupSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled     *bool   `json:"enabled"`
		TransportID *string `json:"transport_id"`
		IdleMinutes *int    `json:"idle_minutes"`
		Retention   *struct {
			Enabled           *bool `json:"enabled"`
			KeepRecentDays    *int  `json:"keep_recent_days"`
			KeepDailyDays     *int  `json:"keep_daily_days"`
			KeepWeeklyMonths  *int  `json:"keep_weekly_months"`
			KeepMonthlyMonths *int  `json:"keep_monthly_months"`
		} `json:"retention"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Enabled != nil {
		v := "0"
		if *req.Enabled {
			v = "1"
		}
		if err := s.settings.Set("cloud.autobackup.enabled", v); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.TransportID != nil {
		if err := s.settings.Set("cloud.autobackup.transport_id", *req.TransportID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.IdleMinutes != nil {
		mins := *req.IdleMinutes
		if mins < 1 {
			mins = 1
		}
		if err := s.settings.Set("cloud.autobackup.idle_minutes", fmt.Sprintf("%d", mins)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.Retention != nil {
		if req.Retention.Enabled != nil {
			v := "0"
			if *req.Retention.Enabled {
				v = "1"
			}
			if err := s.settings.Set("cloud.autobackup.retention_enabled", v); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		// Each numeric field gets a clamp so the API can't be used to push the
		// floor below "always keep at least N recent days." planPrune still
		// fail-closes if anything slips through invalid, but defending here
		// keeps last_error clean when the user just typed a tiny number.
		setIntClamped := func(key string, p *int, min, max int) error {
			if p == nil {
				return nil
			}
			v := *p
			if v < min {
				v = min
			}
			if v > max {
				v = max
			}
			return s.settings.Set(key, fmt.Sprintf("%d", v))
		}
		if err := setIntClamped("cloud.autobackup.retention_keep_recent_days", req.Retention.KeepRecentDays, 1, 365); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := setIntClamped("cloud.autobackup.retention_keep_daily_days", req.Retention.KeepDailyDays, 1, 365); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := setIntClamped("cloud.autobackup.retention_keep_weekly_months", req.Retention.KeepWeeklyMonths, 0, 60); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := setIntClamped("cloud.autobackup.retention_keep_monthly_months", req.Retention.KeepMonthlyMonths, 0, 600); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/cloud/autobackup/trigger
func (s *Server) handleAutoBackupTrigger(w http.ResponseWriter, r *http.Request) {
	if s.autoBackup == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-backup not initialised")
		return
	}
	s.autoBackup.TriggerNow()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
