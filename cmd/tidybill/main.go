package main

import (
	"fmt"
	"os"

	"github.com/adamSHA256/tidybill/internal/cli"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

func main() {

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("app.error_config", err))
		os.Exit(1)
	}

	// Connect to database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("app.error_database", err))
		os.Exit(1)
	}
	defer db.Close()

	// Start CLI
	app := cli.New(db, cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("app.error_general", err))
		os.Exit(1)
	}
}
