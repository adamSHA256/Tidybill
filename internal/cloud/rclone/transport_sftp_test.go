package rclone

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

// TestTransportSFTPE2E exercises Upload/List/Download/Delete through an
// actual SFTP backend. Gated on env vars so CI doesn't try to dial a server
// that doesn't exist:
//
//	TIDYBILL_E2E_SFTP=1
//	TIDYBILL_RCLONE_PATH=/path/to/rclone
//	TIDYBILL_SFTP_HOST=127.0.0.1
//	TIDYBILL_SFTP_PORT=2222
//	TIDYBILL_SFTP_USER=tidybill
//	TIDYBILL_SFTP_PASS=testpass
//
// To stand up a matching server:
//
//	docker run --rm -d -p 2222:22 \
//	    --name tidybill-sftp-test \
//	    atmoz/sftp:alpine \
//	    tidybill:testpass:1001
func TestTransportSFTPE2E(t *testing.T) {
	if os.Getenv("TIDYBILL_E2E_SFTP") != "1" {
		t.Skip("set TIDYBILL_E2E_SFTP=1 to run")
	}
	binPath := os.Getenv("TIDYBILL_RCLONE_PATH")
	if binPath == "" {
		t.Skip("set TIDYBILL_RCLONE_PATH to run")
	}
	host := envOr("TIDYBILL_SFTP_HOST", "127.0.0.1")
	port := envOr("TIDYBILL_SFTP_PORT", "2222")
	user := envOr("TIDYBILL_SFTP_USER", "tidybill")
	pass := envOr("TIDYBILL_SFTP_PASS", "testpass")

	mgr := NewManager(binPath)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer mgr.Stop() //nolint:errcheck

	if err := mgr.EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}

	addr, rcUser, rcPass := mgr.Endpoint()
	rc := NewRC(addr, rcUser, rcPass)

	// Configure an SFTP remote. opt.obscure=true tells rclone to obscure the
	// password parameter — same as the production handler does.
	if err := rc.Call(ctx, "config/create", map[string]any{
		"name": "sftp-test",
		"type": "sftp",
		"parameters": map[string]string{
			"host": host,
			"port": port,
			"user": user,
			"pass": pass,
		},
		"opt": map[string]any{"obscure": true, "nonInteractive": true},
	}, nil); err != nil {
		t.Fatalf("config/create: %v", err)
	}

	tmpDir := t.TempDir()
	// bucketPath defaults to TIDYBILL_SFTP_BUCKET (e.g. "upload/TidyBill"
	// when running against atmoz/sftp where the user's home is read-only and
	// only the configured subdir is writable). Production sets bucketPath to
	// "TidyBill" because real-world SFTP setups give the user a writable
	// home — the test variable just lets us point at any writable prefix.
	bucketPath := envOr("TIDYBILL_SFTP_BUCKET", "upload/TidyBill")
	tr := New(mgr, "sftp", "sftp-test", bucketPath, tmpDir)

	content := []byte("tidybill-sftp-test-content-67890")
	filename := "tidybill-backup-2026-04-28.tidybill"

	// Upload — also exercises mkdir of the bucketPath subdir.
	ref, err := tr.Upload(ctx, filename, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if ref.Filename != filename {
		t.Fatalf("ref.Filename = %q, want %q", ref.Filename, filename)
	}
	if ref.Size != int64(len(content)) {
		t.Fatalf("ref.Size = %d, want %d", ref.Size, len(content))
	}

	// List
	blobs, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, b := range blobs {
		if b.Filename == filename {
			if b.Size != int64(len(content)) {
				t.Errorf("List size = %d, want %d", b.Size, len(content))
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("uploaded file not found in List; got %v", blobs)
	}

	// Download
	rcReader, err := tr.Download(ctx, ref)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rcReader.Close()
	got, err := io.ReadAll(rcReader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloaded content mismatch: got %q, want %q", got, content)
	}

	// Status — should be Connected: true with the bucketPath as account label.
	st, err := tr.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.Connected {
		t.Fatalf("Status not connected: %+v", st)
	}

	// Delete
	if err := tr.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// List should no longer contain the file.
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
