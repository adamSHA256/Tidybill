package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// GET /api/cloud/autosync/status
//
// Returns current settings, last-check info, and any pending prompt state so
// the UI can decide whether to surface a conflict modal. The response is a
// snapshot — the UI polls this on app open and on a soft cadence.
func (s *Server) handleAutoSyncStatus(w http.ResponseWriter, r *http.Request) {
	enabled, _ := s.settings.Get("cloud.autosync.enabled")
	intervalStr, _ := s.settings.Get("cloud.autosync.interval_minutes")
	checkOnStartStr, _ := s.settings.Get("cloud.autosync.check_on_start")
	lastCheckAt, _ := s.settings.Get("cloud.autosync.last_check_at")
	lastPulledAt, _ := s.settings.Get("cloud.autosync.last_pulled_at")
	lastError, _ := s.settings.Get("cloud.autosync.last_error")
	pendingProviderID, _ := s.settings.Get("cloud.autosync.pending_provider_id")
	pendingFilename, _ := s.settings.Get("cloud.autosync.pending_filename")
	pendingCloudModifiedAt, _ := s.settings.Get("cloud.autosync.pending_cloud_modified_at")
	skippedProviderID, _ := s.settings.Get("cloud.autosync.last_skipped_provider_id")

	interval := 60
	if n, err := strconv.Atoi(intervalStr); err == nil && n > 0 {
		interval = n
	}

	// Suppress pending if it was already skipped — defensive: clearPending
	// runs on the skip path too, but stale rows shouldn't show a modal.
	if pendingProviderID != "" && pendingProviderID == skippedProviderID {
		pendingProviderID = ""
		pendingFilename = ""
		pendingCloudModifiedAt = ""
	}

	resp := map[string]any{
		"enabled":         enabled == "1",
		"interval_minutes": interval,
		"check_on_start":  checkOnStartStr == "1",
		"last_check_at":   lastCheckAt,
		"last_pulled_at":  lastPulledAt,
		"last_error":      lastError,
		"pending":         nil,
	}
	if s.autoSync != nil {
		resp["last_action"] = s.autoSync.LastResult().Action
	}
	if pendingProviderID != "" {
		resp["pending"] = map[string]string{
			"provider_id":       pendingProviderID,
			"filename":          pendingFilename,
			"cloud_modified_at": pendingCloudModifiedAt,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// PUT /api/cloud/autosync/settings
func (s *Server) handleAutoSyncSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled         *bool `json:"enabled"`
		IntervalMinutes *int  `json:"interval_minutes"`
		CheckOnStart    *bool `json:"check_on_start"`
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
		if err := s.settings.Set("cloud.autosync.enabled", v); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.IntervalMinutes != nil {
		mins := *req.IntervalMinutes
		if mins < 1 {
			mins = 1
		}
		if err := s.settings.Set("cloud.autosync.interval_minutes", fmt.Sprintf("%d", mins)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.CheckOnStart != nil {
		v := "0"
		if *req.CheckOnStart {
			v = "1"
		}
		if err := s.settings.Set("cloud.autosync.check_on_start", v); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/cloud/autosync/check
//
// Runs a check immediately and returns the result. Does NOT auto-pull on
// "auto_pull" actions even when local is clean — the manual button always
// hands the decision back to the UI so the user sees what happened.
func (s *Server) handleAutoSyncCheck(w http.ResponseWriter, r *http.Request) {
	if s.autoSync == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-sync not initialised")
		return
	}
	res := s.autoSync.Check(r.Context())
	writeJSON(w, http.StatusOK, res)
}

// POST /api/cloud/autosync/pull
//
// Body: { "provider_id": "..." }
//
// Downloads + imports the named cloud blob. Used by the conflict-prompt
// "Just import" button after the user has chosen to overwrite local data.
func (s *Server) handleAutoSyncPull(w http.ResponseWriter, r *http.Request) {
	if s.autoSync == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-sync not initialised")
		return
	}
	var req struct {
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ProviderID == "" {
		writeError(w, http.StatusBadRequest, "provider_id required")
		return
	}
	if err := s.autoSync.PullProviderID(r.Context(), req.ProviderID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/cloud/autosync/skip
//
// Body: { "provider_id": "..." }
//
// Records the user's "Skip" decision so the same blob doesn't re-prompt.
// The marker auto-clears when a newer blob appears.
func (s *Server) handleAutoSyncSkip(w http.ResponseWriter, r *http.Request) {
	if s.autoSync == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-sync not initialised")
		return
	}
	var req struct {
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ProviderID == "" {
		writeError(w, http.StatusBadRequest, "provider_id required")
		return
	}
	s.autoSync.SkipProviderID(req.ProviderID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
