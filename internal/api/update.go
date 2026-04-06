package api

import (
	"net/http"

	"github.com/adamSHA256/tidybill/internal/update"
)

func (s *Server) getUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		writeError(w, http.StatusServiceUnavailable, "update checker not available")
		return
	}

	// Try a non-forced check: returns cache if fresh, fetches if cooldown elapsed
	result, err := s.updater.Check(false)
	if err == nil && result != nil {
		writeJSON(w, http.StatusOK, result)
		return
	}

	// Check failed or disabled — fall back to cached result
	cached := s.updater.GetCached()
	if cached != nil {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	// No cached result — check is disabled or hasn't run yet
	writeJSON(w, http.StatusOK, &update.Result{
		Available:  false,
		CurrentVer: "",
		LatestVer:  "",
	})
}

func (s *Server) postUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		writeError(w, http.StatusServiceUnavailable, "update checker not available")
		return
	}

	result, err := s.updater.Check(true) // force=true for manual check
	if err != nil {
		writeError(w, http.StatusBadGateway, "update check failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
