package rclone

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestManagerE2E(t *testing.T) {
	if os.Getenv("TIDYBILL_E2E_RCLONE") != "1" {
		t.Skip("set TIDYBILL_E2E_RCLONE=1 to run")
	}
	binPath := os.Getenv("TIDYBILL_RCLONE_PATH")
	if binPath == "" {
		t.Skip("set TIDYBILL_RCLONE_PATH to run")
	}

	mgr := NewManager(binPath)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mgr.EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}

	addr, user, pass := mgr.Endpoint()
	if addr == "" || user == "" || pass == "" {
		t.Fatalf("Endpoint returned empty values: addr=%q user=%q pass=%q", addr, user, pass)
	}

	// Verify process is running
	if mgr.cmd == nil || mgr.cmd.Process == nil {
		t.Fatal("cmd.Process is nil after EnsureRunning")
	}
	pid := mgr.cmd.Process.Pid
	if !isRunning(pid) {
		t.Fatalf("process %d is not running after EnsureRunning", pid)
	}

	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Verify process stopped
	if isRunning(pid) {
		t.Fatalf("process %d still running after Stop", pid)
	}
}
