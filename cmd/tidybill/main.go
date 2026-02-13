package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/adamSHA256/tidybill/internal/api"
	"github.com/adamSHA256/tidybill/internal/cli"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

func main() {
	gui := flag.Bool("gui", false, "Start web UI mode (HTTP server)")
	port := flag.String("port", "8080", "HTTP server port (used with --gui)")
	flag.Parse()

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

	if *gui {
		// Web UI mode
		srv := api.NewServer(db.DB, cfg)
		addr := ":" + *port

		log.Printf("TidyBill API server running on http://localhost%s", addr)
		log.Println("  Press Ctrl+C to stop")

		if err := http.ListenAndServe(addr, srv.Router()); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	} else {
		// CLI mode (unchanged)
		app := cli.New(db, cfg)
		if err := app.Run(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.Tf("app.error_general", err))
			os.Exit(1)
		}
	}
}
