package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const Version = "0.2.0"

type Config struct {
	DataDir    string
	DBPath     string
	PDFDir     string
	LogoDir    string
	ExportDir  string
	PreviewDir string
}

func New() (*Config, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	pdfDir, err := getDefaultPDFDir()
	if err != nil {
		pdfDir = filepath.Join(dataDir, "pdfs")
	}

	cfg := &Config{
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "invoices.db"),
		PDFDir:     pdfDir,
		LogoDir:    filepath.Join(dataDir, "logos"),
		ExportDir:  filepath.Join(dataDir, "exports"),
		PreviewDir: filepath.Join(dataDir, "previews"),
	}

	dirs := []string{cfg.DataDir, cfg.PDFDir, cfg.LogoDir, cfg.ExportDir, cfg.PreviewDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// NewWithDataDir creates a Config using the specified data directory.
// Used by the mobile app where the data directory is provided by the platform.
func NewWithDataDir(dataDir string) (*Config, error) {
	cfg := &Config{
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "invoices.db"),
		PDFDir:     filepath.Join(dataDir, "pdfs"),
		LogoDir:    filepath.Join(dataDir, "logos"),
		ExportDir:  filepath.Join(dataDir, "exports"),
		PreviewDir: filepath.Join(dataDir, "previews"),
	}

	dirs := []string{cfg.DataDir, cfg.PDFDir, cfg.LogoDir, cfg.ExportDir, cfg.PreviewDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func getDataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "TidyBill")

	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Application Support", "TidyBill")

	default:
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config")
		}
		baseDir = filepath.Join(configDir, "tidybill")
	}

	return baseDir, nil
}

// ApplySettings reads directory overrides from the settings table and applies them.
// It creates directories if they don't exist.
func (c *Config) ApplySettings(get func(key string) (string, error)) error {
	dirKeys := map[string]*string{
		"dir.logos":    &c.LogoDir,
		"dir.pdfs":     &c.PDFDir,
		"dir.previews": &c.PreviewDir,
	}

	for key, field := range dirKeys {
		val, err := get(key)
		if err != nil {
			return fmt.Errorf("reading setting %s: %w", key, err)
		}
		if val != "" {
			*field = val
		}
	}

	// Ensure all directories exist
	for _, dir := range []string{c.LogoDir, c.PDFDir, c.PreviewDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return nil
}

func getDefaultPDFDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// On Linux, check XDG_DOCUMENTS_DIR first
	if runtime.GOOS == "linux" {
		if docsDir := os.Getenv("XDG_DOCUMENTS_DIR"); docsDir != "" {
			return filepath.Join(docsDir, "TidyBill"), nil
		}
	}

	return filepath.Join(home, "Documents", "TidyBill"), nil
}

func (c *Config) GetPDFPath(invoiceNumber string, year int, supplierName string) string {
	supplierDir := sanitizeDirName(supplierName)
	yearDir := filepath.Join(c.PDFDir, supplierDir, fmt.Sprintf("%d", year))
	os.MkdirAll(yearDir, 0755)
	return filepath.Join(yearDir, invoiceNumber+".pdf")
}

func sanitizeDirName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	result := strings.TrimSpace(replacer.Replace(name))
	if result == "" {
		return "default"
	}
	return result
}
