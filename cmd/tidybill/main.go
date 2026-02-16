package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adamSHA256/tidybill/internal/api"
	"github.com/adamSHA256/tidybill/internal/cli"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/i18n"
)

func main() {
	gui := flag.Bool("gui", false, "Start web UI mode (HTTP server)")
	port := flag.String("port", "8080", "HTTP server port (used with --gui)")
	parentPID := flag.Int("parent-pid", 0, "Parent process PID (exit when parent dies)")
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

	// Apply directory settings from DB
	settings := repository.NewSettingsRepository(db.DB)
	if err := cfg.ApplySettings(settings.Get); err != nil {
		log.Printf("Warning: failed to apply settings: %v", err)
	}

	if *gui {
		// Web UI mode
		srv := api.NewServer(db.DB, cfg)

		listener, err := net.Listen("tcp", ":"+*port)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
		actualPort := listener.Addr().(*net.TCPAddr).Port
		fmt.Printf("TIDYBILL_PORT=%d\n", actualPort)

		httpServer := &http.Server{Handler: srv.Router()}

		// Shutdown on SIGINT/SIGTERM
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		// Watch parent process — exit if it dies (covers crashes)
		if *parentPID > 0 {
			go func() {
				for {
					time.Sleep(2 * time.Second)
					if err := syscall.Kill(*parentPID, 0); err != nil {
						log.Println("[tidybill] parent process gone, shutting down")
						httpServer.Shutdown(context.Background())
						return
					}
				}
			}()
		}

		// Start server in background
		go func() {
			log.Printf("TidyBill API server running on http://localhost:%d", actualPort)
			if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed: %v", err)
			}
		}()

		// Block until signal or parent death triggers shutdown
		<-ctx.Done()
		log.Println("[tidybill] shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	} else {
		// CLI mode (unchanged)
		app := cli.New(db, cfg)
		if err := app.Run(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.Tf("app.error_general", err))
			os.Exit(1)
		}
	}
}
