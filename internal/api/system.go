package api

import (
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/adamSHA256/tidybill/internal/config"
)

func (s *Server) getFirstRun(w http.ResponseWriter, r *http.Request) {
	suppliers, err := s.suppliers.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{
		"first_run": len(suppliers) == 0,
	})
}

func (s *Server) getLocale(w http.ResponseWriter, r *http.Request) {
	lang := detectOSLang()
	writeJSON(w, http.StatusOK, map[string]string{
		"detected_lang": lang,
	})
}

// detectOSLang reads OS environment to detect the user's language.
// Returns "cs", "sk", or "en".
func detectOSLang() string {
	var raw string

	if runtime.GOOS == "windows" {
		// On Windows, try common env vars set by terminals / Git Bash
		for _, key := range []string{"LANG", "LANGUAGE", "LC_ALL"} {
			if v := os.Getenv(key); v != "" {
				raw = v
				break
			}
		}
	} else {
		// Linux / macOS
		for _, key := range []string{"LANG", "LC_ALL", "LANGUAGE"} {
			if v := os.Getenv(key); v != "" {
				raw = v
				break
			}
		}
	}

	if raw == "" {
		return "en"
	}

	// raw is something like "cs_CZ.UTF-8" or "en_US.UTF-8"
	// Extract the two-letter language code
	raw = strings.ToLower(raw)
	// Remove encoding suffix
	if idx := strings.Index(raw, "."); idx > 0 {
		raw = raw[:idx]
	}
	// Take just the language part before underscore
	if idx := strings.Index(raw, "_"); idx > 0 {
		raw = raw[:idx]
	}

	switch raw {
	case "cs":
		return "cs"
	case "sk":
		return "sk"
	default:
		return "en"
	}
}

func (s *Server) getAbout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":           config.Version,
		"description":       "Simple, private invoicing for freelancers and small businesses.",
		"github_issues_url": "https://github.com/adamSHA256/tidybill/issues",
		"monero_address":    "<42GYPXCvn42NbsujN8wRVrQj4xLnWubXm4BJmiyZjkb8PuGCMb75iC96BHkia6LJM57BfVqyGJm2VH3Mr97c269hRxSidqG>",
		"bitcoin_address":   "<bc1q5dqj6jfpuq47qvmu2w3awt7jz4zlyyfcyreayx>",
	})
}
