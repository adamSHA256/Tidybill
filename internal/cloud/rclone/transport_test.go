package rclone

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestTransportE2E(t *testing.T) {
	if os.Getenv("TIDYBILL_E2E_RCLONE") != "1" {
		t.Skip("set TIDYBILL_E2E_RCLONE=1 to run")
	}
	binPath := os.Getenv("TIDYBILL_RCLONE_PATH")
	if binPath == "" {
		t.Skip("set TIDYBILL_RCLONE_PATH to run")
	}

	mgr := NewManager(binPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer mgr.Stop() //nolint:errcheck

	if err := mgr.EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}

	// Create an in-memory local remote pointed at a temp dir.
	localRoot := t.TempDir()
	addr, user, pass := mgr.Endpoint()
	rc := NewRC(addr, user, pass)

	if err := rc.Call(ctx, "config/create", map[string]any{
		"name": "local-test",
		"type": "local",
		"parameters": map[string]string{
			"root": localRoot,
		},
		"opt": map[string]any{"nonInteractive": true},
	}, nil); err != nil {
		t.Fatalf("config/create: %v", err)
	}

	tmpDir := t.TempDir()
	tr := New(mgr, "local", "local-test", "", tmpDir)

	content := []byte("tidybill-test-content-12345")
	filename := "tidybill-backup-2026-01-01.tidybill"

	// Upload
	ref, err := tr.Upload(ctx, filename, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if ref.Filename != filename {
		t.Fatalf("ref.Filename = %q, want %q", ref.Filename, filename)
	}

	// List
	blobs, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, b := range blobs {
		if b.Filename == filename {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("uploaded file not found in List; got %v", blobs)
	}

	// Download
	rc2, err := tr.Download(ctx, ref)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rc2.Close()
	got, err := io.ReadAll(rc2)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloaded content mismatch: got %q, want %q", got, content)
	}

	// Delete
	if err := tr.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// List should now be empty
	blobs, err = tr.List(ctx)
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	for _, b := range blobs {
		if b.Filename == filename {
			t.Fatalf("file still present in List after Delete")
		}
	}
}
