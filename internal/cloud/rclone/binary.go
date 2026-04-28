package rclone

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// LocateBinary returns the absolute path of the bundled rclone binary.
// Tauri copies sidecars into the resources directory next to the main
// executable at runtime. Packager-specific install layouts (.deb/.rpm)
// can place resources under /usr/lib/tidybill/, so the candidate list
// includes that too. TIDYBILL_RCLONE_PATH overrides everything.
func LocateBinary() (string, error) {
	if explicit := os.Getenv("TIDYBILL_RCLONE_PATH"); explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit, nil
		}
	}
	exeDir, err := selfDir()
	if err != nil {
		return "", err
	}
	name := "rclone"
	if runtime.GOOS == "windows" {
		name = "rclone.exe"
	}
	candidates := []string{
		filepath.Join(exeDir, name),                    // default Tauri layout
		filepath.Join(exeDir, "resources", name),       // alt Tauri layout
		filepath.Join(exeDir, "..", "Resources", name), // macOS .app bundle
	}
	// Linux distro-install fallbacks. Harmless on other OSes.
	if runtime.GOOS == "linux" {
		candidates = append(candidates,
			"/usr/lib/tidybill/"+name,
			"/usr/local/lib/tidybill/"+name,
			"/opt/tidybill/"+name,
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("rclone binary not found (searched %v). Packagers may need to set TIDYBILL_RCLONE_PATH.", candidates)
}

func selfDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
