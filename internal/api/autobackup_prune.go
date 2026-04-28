package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

// PruneConfig captures the four GFS knobs from settings. Values are in their
// natural unit (days for daily-bucket boundaries, months for weekly/monthly).
type PruneConfig struct {
	KeepRecentDays    int
	KeepDailyDays     int
	KeepWeeklyMonths  int
	KeepMonthlyMonths int // 0 = keep monthly backups forever
}

// valid returns true when the config can safely drive a prune. A failed check
// causes the runner to skip pruning entirely (fail closed) — better to leave
// extra files than to delete on a misconfigured policy.
func (c PruneConfig) valid() bool {
	if c.KeepRecentDays < 1 {
		return false
	}
	if c.KeepDailyDays < c.KeepRecentDays {
		return false
	}
	if c.KeepWeeklyMonths < 0 {
		return false
	}
	if c.KeepMonthlyMonths < 0 {
		return false
	}
	return true
}

// planPrune is the pure planner: given the current list of cloud blobs and a
// reference "now", returns the subset to delete according to the GFS schedule.
//
// Algorithm:
//  1. Filter out anything that isn't a tidybill-backup-*.tidybill file.
//  2. Sort newest-first.
//  3. ALWAYS keep entries[0] (most recent), as an absolute floor — even if the
//     buckets below would somehow exclude it.
//  4. Walk newest → oldest, classify each by age:
//     - within KeepRecentDays              → keep all
//     - within KeepDailyDays               → keep most recent per UTC day
//     - within KeepWeeklyMonths months     → keep most recent per ISO week
//     - older                              → keep most recent per month, unless
//                                            KeepMonthlyMonths > 0 and the
//                                            backup is older than that horizon
//                                            (in which case it's deleted)
//
// Anything not in the keep set is returned in toDelete.
func planPrune(blobs []cloud.BlobRef, now time.Time, cfg PruneConfig) []cloud.BlobRef {
	if !cfg.valid() {
		return nil
	}

	type entry struct {
		ref cloud.BlobRef
		ts  time.Time
	}
	var entries []entry
	for i := range blobs {
		b := blobs[i]
		if !isTidybillBackupFilename(b.Filename) {
			continue
		}
		ts := blobTime(&b)
		if ts.IsZero() {
			continue
		}
		entries = append(entries, entry{ref: b, ts: ts})
	}
	if len(entries) == 0 {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ts.After(entries[j].ts)
	})

	keep := make(map[string]struct{}, len(entries))
	keep[entries[0].ref.ID] = struct{}{} // absolute floor

	recentBoundary := now.AddDate(0, 0, -cfg.KeepRecentDays)
	dailyBoundary := now.AddDate(0, 0, -cfg.KeepDailyDays)
	weeklyBoundary := now.AddDate(0, -cfg.KeepWeeklyMonths, 0)
	hasMonthlyHorizon := cfg.KeepMonthlyMonths > 0
	monthlyHorizon := time.Time{}
	if hasMonthlyHorizon {
		monthlyHorizon = now.AddDate(0, -cfg.KeepMonthlyMonths, 0)
	}

	seenDays := map[string]bool{}
	seenWeeks := map[string]bool{}
	seenMonths := map[string]bool{}

	for _, e := range entries {
		switch {
		case e.ts.After(recentBoundary):
			keep[e.ref.ID] = struct{}{}
		case e.ts.After(dailyBoundary):
			day := e.ts.UTC().Format("2006-01-02")
			if !seenDays[day] {
				keep[e.ref.ID] = struct{}{}
				seenDays[day] = true
			}
		case e.ts.After(weeklyBoundary):
			y, w := e.ts.UTC().ISOWeek()
			key := fmt.Sprintf("%d-%02d", y, w)
			if !seenWeeks[key] {
				keep[e.ref.ID] = struct{}{}
				seenWeeks[key] = true
			}
		default:
			// Bucket D (monthly).
			if hasMonthlyHorizon && !e.ts.After(monthlyHorizon) {
				continue // older than the monthly horizon → delete
			}
			month := e.ts.UTC().Format("2006-01")
			if !seenMonths[month] {
				keep[e.ref.ID] = struct{}{}
				seenMonths[month] = true
			}
		}
	}

	var toDelete []cloud.BlobRef
	for _, e := range entries {
		if _, ok := keep[e.ref.ID]; !ok {
			toDelete = append(toDelete, e.ref)
		}
	}
	return toDelete
}

func isTidybillBackupFilename(name string) bool {
	return strings.HasPrefix(name, "tidybill-backup-") && strings.HasSuffix(name, ".tidybill")
}

// pruneBackups runs after a successful auto-backup upload. It re-lists the
// remote, applies planPrune, and deletes the resulting set. Each deletion is
// logged with full provider_id + filename so the user can audit if needed.
//
// Errors from individual deletes are logged but don't abort the whole run —
// network blips on one file shouldn't block cleanup of the rest.
func (a *AutoBackupService) pruneBackups(ctx context.Context, t cloud.Transport) error {
	enabled, _ := a.settings.Get("cloud.autobackup.retention_enabled")
	if enabled != "1" {
		return nil
	}

	cfg, err := a.loadPruneConfig()
	if err != nil {
		log.Printf("autobackup-prune: skipping (invalid config): %v", err)
		return nil // fail closed
	}

	listCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	blobs, err := t.List(listCtx)
	if err != nil {
		return fmt.Errorf("list before prune: %w", err)
	}

	toDelete := planPrune(blobs, time.Now().UTC(), cfg)
	if len(toDelete) == 0 {
		return nil
	}

	for _, b := range toDelete {
		delCtx, delCancel := context.WithTimeout(ctx, 30*time.Second)
		err := t.Delete(delCtx, b)
		delCancel()
		if err != nil {
			log.Printf("autobackup-prune: delete %s (%s) failed: %v", b.ID, b.Filename, err)
			continue
		}
		log.Printf("autobackup-prune: deleted %s (%s)", b.ID, b.Filename)
	}
	return nil
}

func (a *AutoBackupService) loadPruneConfig() (PruneConfig, error) {
	get := func(key string) (int, error) {
		v, _ := a.settings.Get(key)
		if v == "" {
			return 0, fmt.Errorf("missing setting %s", key)
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("setting %s: %w", key, err)
		}
		return n, nil
	}
	keepRecent, err := get("cloud.autobackup.retention_keep_recent_days")
	if err != nil {
		return PruneConfig{}, err
	}
	keepDaily, err := get("cloud.autobackup.retention_keep_daily_days")
	if err != nil {
		return PruneConfig{}, err
	}
	keepWeekly, err := get("cloud.autobackup.retention_keep_weekly_months")
	if err != nil {
		return PruneConfig{}, err
	}
	keepMonthly, err := get("cloud.autobackup.retention_keep_monthly_months")
	if err != nil {
		return PruneConfig{}, err
	}
	cfg := PruneConfig{
		KeepRecentDays:    keepRecent,
		KeepDailyDays:     keepDaily,
		KeepWeeklyMonths:  keepWeekly,
		KeepMonthlyMonths: keepMonthly,
	}
	if !cfg.valid() {
		return PruneConfig{}, errors.New("retention config out of range")
	}
	return cfg, nil
}
