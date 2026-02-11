package main

import (
	"fmt"
	"os"
	"github.com/user/invoice-app/internal/cli"
	"github.com/user/invoice-app/internal/config"
	"github.com/user/invoice-app/internal/database"
)

func main() {

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chyba konfigurace: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chyba databáze: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Start CLI
	app := cli.New(db, cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Chyba: %v\n", err)
		os.Exit(1)
	}
}
