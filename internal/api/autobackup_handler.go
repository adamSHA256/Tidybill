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
	})
}

// PUT /api/cloud/autobackup/settings
func (s *Server) handleAutoBackupSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled     *bool   `json:"enabled"`
		TransportID *string `json:"transport_id"`
		IdleMinutes *int    `json:"idle_minutes"`
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
