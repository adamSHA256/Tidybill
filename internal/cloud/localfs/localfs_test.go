package localfs

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tr := New(dir)
	ctx := context.Background()

	content := []byte("tidybill test content")
	filename := "tidybill-backup-2026-01-01.tidybill"

	// Upload
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
	if len(blobs) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(blobs))
	}

	// Download
	rc, err := tr.Download(ctx, ref)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloaded content mismatch")
	}

	// Delete
	if err := tr.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// List now empty
	blobs, err = tr.List(ctx)
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(blobs) != 0 {
		t.Fatalf("expected 0 blobs after delete, got %d", len(blobs))
	}
}

func TestUploadCreatesDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "export", "dir")
	tr := New(dir)
	ctx := context.Background()

	content := []byte("data")
	_, err := tr.Upload(ctx, "tidybill-backup-2026-01-02.tidybill", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Upload to non-existent dir: %v", err)
	}
}

func TestPathEscapeDownload(t *testing.T) {
	dir := t.TempDir()
	tr := New(dir)
	ctx := context.Background()

	escapingRef := cloud.BlobRef{ID: "../../etc/passwd"}
	_, err := tr.Download(ctx, escapingRef)
	if err == nil {
		t.Fatal("expected error for path escape, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to read outside") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathEscapeDelete(t *testing.T) {
	dir := t.TempDir()
	tr := New(dir)
	ctx := context.Background()

	escapingRef := cloud.BlobRef{ID: "../../etc/passwd"}
	err := tr.Delete(ctx, escapingRef)
	if err == nil {
		t.Fatal("expected error for path escape, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to delete outside") {
		t.Fatalf("unexpected error: %v", err)
	}
}
