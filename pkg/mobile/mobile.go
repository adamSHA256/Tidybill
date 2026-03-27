package mobile

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/adamSHA256/tidybill/internal/api"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

var (
	mu     sync.Mutex
	server *http.Server
	db     *database.DB
)

// StartServer starts the TidyBill HTTP API server.
// dataDir is the app's internal data directory (e.g. Android filesDir).
// Returns the port number the server is listening on.
func StartServer(dataDir string) (int, error) {
	mu.Lock()
	defer mu.Unlock()

	if server != nil {
		return 0, fmt.Errorf("server already running")
	}

	cfg, err := config.NewWithDataDir(dataDir)
	if err != nil {
		return 0, fmt.Errorf("config: %w", err)
	}

	db, err = database.New(cfg.DBPath)
	if err != nil {
		return 0, fmt.Errorf("database: %w", err)
	}

	// Apply settings from database (directory overrides, language, etc.).
	// Ignore errors: first run has no settings yet.
	settingsRepo := repository.NewSettingsRepository(db.DB)
	_ = cfg.ApplySettings(settingsRepo.Get)

	// Set language from settings
	if lang, err := settingsRepo.Get("language"); err == nil && lang != "" {
		switch lang {
		case "cs":
			i18n.SetLang(i18n.CS)
		case "sk":
			i18n.SetLang(i18n.SK)
		case "en":
			i18n.SetLang(i18n.EN)
		}
	}

	// Persist email template defaults if not already set
	settingsRepo.SetDefault("email.default_subject", "Faktura ((number))")
	settingsRepo.SetDefault("email.default_body", "Dobrý den,\n\nv příloze zasílám fakturu č. ((number)) na částku ((total)).\nSplatnost: ((due_date)).\n\nS pozdravem\n((supplier))")
	settingsRepo.SetDefault("email.copy_subject", "TidyBill - ((subject))")

	srv := api.NewServer(db.DB, cfg)

	listener, err := net.Listen("tcp", "127.0.0.1:18080")
	if err != nil {
		db.Close()
		return 0, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	server = &http.Server{Handler: srv.Router()}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("tidybill server error: %v\n", err)
		}
	}()

	return port, nil
}

// StopServer gracefully shuts down the server.
func StopServer() {
	mu.Lock()
	defer mu.Unlock()

	if server != nil {
		server.Shutdown(context.Background())
		server = nil
	}
	if db != nil {
		db.Close()
		db = nil
	}
}
