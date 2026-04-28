package api

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

// defaultCfg matches the migration defaults so tests describe real behaviour.
var defaultCfg = PruneConfig{
	KeepRecentDays:    7,
	KeepDailyDays:     30,
	KeepWeeklyMonths:  6,
	KeepMonthlyMonths: 0,
}

// blobAt builds a synthetic backup BlobRef with the canonical filename and
// an explicit ModTime, so tests don't depend on filename-parsing as well.
func blobAt(id string, ts time.Time) cloud.BlobRef {
	return cloud.BlobRef{
		ID:         id,
		Filename:   "tidybill-backup-" + ts.UTC().Format("2006-01-02T15-04-05Z") + ".tidybill",
		ModifiedAt: ts.UTC(),
	}
}

func deletedIDs(toDelete []cloud.BlobRef) []string {
	ids := make([]string, 0, len(toDelete))
	for _, b := range toDelete {
		ids = append(ids, b.ID)
	}
	sort.Strings(ids)
	return ids
}

func TestPlanPrune_EmptyList(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	got := planPrune(nil, now, defaultCfg)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestPlanPrune_OnlyRecent_KeepsAll(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	blobs := []cloud.BlobRef{
		blobAt("a", now.Add(-1*time.Hour)),
		blobAt("b", now.Add(-24*time.Hour)),
		blobAt("c", now.Add(-3*24*time.Hour)),
		blobAt("d", now.Add(-6*24*time.Hour)),
	}
	got := planPrune(blobs, now, defaultCfg)
	if len(got) != 0 {
		t.Errorf("expected 0 deletes, got %v", deletedIDs(got))
	}
}

func TestPlanPrune_OnlyOld_KeepsMonthlyAndMostRecent(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	// All timestamps are >6 months old → bucket D (monthly).
	// With KeepMonthlyMonths=0 (forever) we keep one per month.
	blobs := []cloud.BlobRef{
		blobAt("jan-1", time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),
		blobAt("jan-2", time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)),
		blobAt("feb-1", time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)),
		blobAt("mar-1", time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
		blobAt("mar-2", time.Date(2025, 3, 28, 0, 0, 0, 0, time.UTC)),
	}
	got := deletedIDs(planPrune(blobs, now, defaultCfg))
	want := []string{"jan-1", "mar-1"} // older within each month deleted
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlanPrune_Mixed_GFSBuckets(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	blobs := []cloud.BlobRef{
		// Bucket A (last 7d) — keep all.
		blobAt("recent-1", now.Add(-1*time.Hour)),
		blobAt("recent-2", now.Add(-2*24*time.Hour)),
		blobAt("recent-3", now.Add(-6*24*time.Hour)),
		// Bucket B (7-30d) — keep most recent per UTC day.
		blobAt("day-10a", now.Add(-10*24*time.Hour-1*time.Hour)),
		blobAt("day-10b", now.Add(-10*24*time.Hour-5*time.Hour)),
		blobAt("day-15", now.Add(-15*24*time.Hour)),
		// Bucket C (30d-6mo) — keep most recent per ISO week.
		blobAt("week-A1", time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)),  // ISO week 2026-W06
		blobAt("week-A2", time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)),  // same week, newer
		blobAt("week-B1", time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)), // 2026-W04
		// Bucket D (>6mo) — keep monthly.
		blobAt("month-A1", time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)),
		blobAt("month-A2", time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC)),
	}
	got := deletedIDs(planPrune(blobs, now, defaultCfg))
	want := []string{"day-10b", "month-A1", "week-A1"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlanPrune_ExactlyOnePerDay_NoDeletes(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	var blobs []cloud.BlobRef
	for d := 8; d <= 25; d++ {
		blobs = append(blobs, blobAt(fmt.Sprintf("d%d", d), now.Add(-time.Duration(d)*24*time.Hour)))
	}
	got := planPrune(blobs, now, defaultCfg)
	if len(got) != 0 {
		t.Errorf("expected 0 deletes (one per day), got %v", deletedIDs(got))
	}
}

func TestPlanPrune_MultiplePerDay_KeepsMostRecent(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	// All 4 timestamps on UTC day 2026-04-18 (10 days old → bucket B).
	blobs := []cloud.BlobRef{
		blobAt("morning",  time.Date(2026, 4, 18, 8, 0, 0, 0, time.UTC)),
		blobAt("midday",   time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)),
		blobAt("evening",  time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC)),
		blobAt("midnight", time.Date(2026, 4, 18, 23, 30, 0, 0, time.UTC)),
	}
	got := deletedIDs(planPrune(blobs, now, defaultCfg))
	// Only the latest of the day survives.
	want := []string{"evening", "midday", "morning"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlanPrune_MultiplePerWeek_KeepsMostRecent(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	// All in 2026-W06 (Mon Feb 2 .. Sun Feb 8) — within 30d-6mo window.
	blobs := []cloud.BlobRef{
		blobAt("w06-mon", time.Date(2026, 2, 2, 8, 0, 0, 0, time.UTC)),
		blobAt("w06-wed", time.Date(2026, 2, 4, 8, 0, 0, 0, time.UTC)),
		blobAt("w06-sun", time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)),
	}
	got := deletedIDs(planPrune(blobs, now, defaultCfg))
	want := []string{"w06-mon", "w06-wed"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlanPrune_SkipsNonTidybillFiles(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	blobs := []cloud.BlobRef{
		{ID: "junk-old", Filename: "random.txt", ModifiedAt: old},
		blobAt("backup-old", old),
	}
	got := planPrune(blobs, now, defaultCfg)
	for _, b := range got {
		if b.ID == "junk-old" {
			t.Errorf("planPrune wanted to delete a non-tidybill file: %s", b.Filename)
		}
	}
}

func TestPlanPrune_AlwaysKeepsMostRecentEvenWhenAggressive(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	cfg := PruneConfig{
		KeepRecentDays:    1,
		KeepDailyDays:     2,
		KeepWeeklyMonths:  1,
		KeepMonthlyMonths: 1, // delete monthly older than 1 month
	}
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // way older than monthly horizon
	blobs := []cloud.BlobRef{blobAt("only-old", old)}
	got := planPrune(blobs, now, cfg)
	if len(got) != 0 {
		t.Errorf("expected to keep the sole backup as floor, got delete %v", deletedIDs(got))
	}
}

func TestPlanPrune_FailClosed_OnInvalidConfig(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	blobs := []cloud.BlobRef{
		blobAt("recent", now.Add(-1*time.Hour)),
		blobAt("old", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	bad := PruneConfig{KeepRecentDays: 0} // invalid — must be >= 1
	got := planPrune(blobs, now, bad)
	if got != nil {
		t.Errorf("expected nil (fail-closed) on invalid config, got %v", deletedIDs(got))
	}
}

func TestPlanPrune_MonthlyHorizon_DeletesPastIt(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	cfg := defaultCfg
	cfg.KeepMonthlyMonths = 12 // keep monthly only for the last 12 months
	// 12-month horizon from 2026-04-28 → 2025-04-28. Anything older = delete.
	blobs := []cloud.BlobRef{
		blobAt("2024-jan", time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)),  // > 12mo old → delete
		blobAt("2025-jan", time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),  // > 12mo old → delete
		blobAt("2025-jul", time.Date(2025, 7, 5, 0, 0, 0, 0, time.UTC)),  // < 12mo old, monthly → keep
		blobAt("recent", now.Add(-2*time.Hour)),                           // bucket A → keep
	}
	got := deletedIDs(planPrune(blobs, now, cfg))
	want := []string{"2024-jan", "2025-jan"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
