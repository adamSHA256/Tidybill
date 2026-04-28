package api

import (
	"testing"
	"time"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

func TestParseBackupFilenameTime(t *testing.T) {
	cases := []struct {
		filename string
		want     time.Time
		zero     bool
	}{
		{"tidybill-backup-2026-04-28T14-30-45Z.tidybill", time.Date(2026, 4, 28, 14, 30, 45, 0, time.UTC), false},
		{"tidybill-backup-2026-01-01T00-00-00Z.tidybill", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"some-other-file.tidybill", time.Time{}, true},
		{"tidybill-backup-no-timestamp.tidybill", time.Time{}, true},
		{"tidybill-backup-2026-04-28T14-30-45Z.txt", time.Time{}, true},
		{"tidybill-backup-bogus.tidybill", time.Time{}, true},
	}
	for _, c := range cases {
		got := parseBackupFilenameTime(c.filename)
		if c.zero {
			if !got.IsZero() {
				t.Errorf("%q: expected zero, got %v", c.filename, got)
			}
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("%q: got %v, want %v", c.filename, got, c.want)
		}
	}
}

func TestPickMostRecentBackup(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		b, _ := pickMostRecentBackup(nil)
		if b != nil {
			t.Errorf("expected nil for empty list, got %+v", b)
		}
	})

	t.Run("uses ModifiedAt when present", func(t *testing.T) {
		blobs := []cloud.BlobRef{
			{ID: "old", Filename: "tidybill-backup-2026-01-01T00-00-00Z.tidybill", ModifiedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{ID: "new", Filename: "tidybill-backup-2026-04-28T14-30-45Z.tidybill", ModifiedAt: time.Date(2026, 4, 28, 14, 30, 45, 0, time.UTC)},
			{ID: "mid", Filename: "tidybill-backup-2026-02-15T10-00-00Z.tidybill", ModifiedAt: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)},
		}
		b, ts := pickMostRecentBackup(blobs)
		if b == nil || b.ID != "new" {
			t.Fatalf("expected new, got %+v", b)
		}
		if !ts.Equal(time.Date(2026, 4, 28, 14, 30, 45, 0, time.UTC)) {
			t.Errorf("unexpected ts %v", ts)
		}
	})

	t.Run("falls back to filename when ModifiedAt is zero", func(t *testing.T) {
		blobs := []cloud.BlobRef{
			{ID: "old", Filename: "tidybill-backup-2026-01-01T00-00-00Z.tidybill"},
			{ID: "new", Filename: "tidybill-backup-2026-04-28T14-30-45Z.tidybill"},
		}
		b, _ := pickMostRecentBackup(blobs)
		if b == nil || b.ID != "new" {
			t.Fatalf("expected new (filename fallback), got %+v", b)
		}
	})

	t.Run("skips non-tidybill files", func(t *testing.T) {
		blobs := []cloud.BlobRef{
			{ID: "junk", Filename: "random.txt", ModifiedAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
			{ID: "ok", Filename: "tidybill-backup-2026-04-28T14-30-45Z.tidybill", ModifiedAt: time.Date(2026, 4, 28, 14, 30, 45, 0, time.UTC)},
		}
		b, _ := pickMostRecentBackup(blobs)
		if b == nil || b.ID != "ok" {
			t.Fatalf("expected ok (junk filtered), got %+v", b)
		}
	})

	t.Run("skips blobs with unparseable filename and zero ModifiedAt", func(t *testing.T) {
		blobs := []cloud.BlobRef{
			{ID: "weird", Filename: "tidybill-backup-bogus.tidybill"},
			{ID: "ok", Filename: "tidybill-backup-2026-04-28T14-30-45Z.tidybill"},
		}
		b, _ := pickMostRecentBackup(blobs)
		if b == nil || b.ID != "ok" {
			t.Fatalf("expected ok, got %+v", b)
		}
	})
}
