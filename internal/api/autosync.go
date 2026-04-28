package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cloud"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
	"github.com/adamSHA256/tidybill/internal/database/repository"
)

// SyncCheckResult is what AutoSyncService.Check returns, exposed verbatim
// over the HTTP status endpoint.
type SyncCheckResult struct {
	// Action is one of:
	//   "none"      — cloud not newer than local-known state, nothing to do
	//   "auto_pull" — cloud is newer AND local is clean since last sync; safe to pull
	//   "prompt"    — cloud is newer AND local has un-synced changes; user must decide
	//   "skipped"   — cloud is newer but matches the user's last "Skip" choice
	//   "error"     — something went wrong (see Message)
	Action           string `json:"action"`
	ProviderID       string `json:"provider_id,omitempty"`
	Filename         string `json:"filename,omitempty"`
	CloudModifiedAt  string `json:"cloud_modified_at,omitempty"`
	LocalModifiedAt  string `json:"local_modified_at,omitempty"`
	LastSyncedAt     string `json:"last_synced_at,omitempty"`
	Message          string `json:"message,omitempty"`
}

// AutoSyncService runs a background goroutine that periodically checks the
// configured cloud target for a backup newer than the local state.
//
// Behaviour matrix:
//   - cloud has nothing newer        → record last_check_at, no action
//   - cloud newer, local clean       → auto-pull (download + import full_replace)
//   - cloud newer, local dirty       → record pending prompt; UI must resolve
//   - cloud newer, but already skipped by user → leave pending state cleared
//
// "Local clean since last sync" means: data.last_write_at <= max(autobackup.last_run_at, autosync.last_pulled_at).
type AutoSyncService struct {
	settings *repository.SettingsRepository
	registry *cloud.Registry
	importer *backup.ImportService
	kc       keychain.Store

	mu      sync.Mutex
	lastRun SyncCheckResult

	trigger chan struct{} // buffered(1): signals an immediate check
	ctx     context.Context
	cancel  context.CancelFunc
}

func newAutoSyncService(
	settings *repository.SettingsRepository,
	registry *cloud.Registry,
	importer *backup.ImportService,
	kc keychain.Store,
) *AutoSyncService {
	ctx, cancel := context.WithCancel(context.Background())
	return &AutoSyncService{
		settings: settings,
		registry: registry,
		importer: importer,
		kc:       kc,
		trigger:  make(chan struct{}, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Shutdown cancels the goroutine. Safe to call multiple times.
func (a *AutoSyncService) Shutdown() {
	a.cancel()
}

// TriggerNow enqueues an immediate check.
func (a *AutoSyncService) TriggerNow() {
	select {
	case a.trigger <- struct{}{}:
	default:
	}
}

// Run loops on a 5-minute cadence (or interval_minutes setting) and calls tick.
// On startup, if cloud.autosync.check_on_start=1 and enabled=1, it kicks one
// immediate check after a small delay so the rclone manager is up.
func (a *AutoSyncService) Run() {
	if a.shouldCheckOnStart() {
		// Delay so rclone manager / remotes have a chance to register.
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(15 * time.Second):
		}
		a.tick()
	}

	for {
		next := a.intervalDuration()
		select {
		case <-a.ctx.Done():
			return
		case <-a.trigger:
			a.tick()
		case <-time.After(next):
			a.tick()
		}
	}
}

func (a *AutoSyncService) shouldCheckOnStart() bool {
	enabled, _ := a.settings.Get("cloud.autosync.enabled")
	if enabled != "1" {
		return false
	}
	v, _ := a.settings.Get("cloud.autosync.check_on_start")
	return v == "1"
}

func (a *AutoSyncService) intervalDuration() time.Duration {
	v, _ := a.settings.Get("cloud.autosync.interval_minutes")
	mins, _ := strconv.Atoi(v)
	if mins < 1 {
		mins = 60
	}
	return time.Duration(mins) * time.Minute
}

// tick is the periodic checker. Errors are recorded but never panic.
func (a *AutoSyncService) tick() {
	enabled, _ := a.settings.Get("cloud.autosync.enabled")
	if enabled != "1" {
		return
	}
	res := a.Check(a.ctx)
	if res.Action == "auto_pull" {
		// Goroutine path: pull immediately when local is clean.
		if err := a.pull(a.ctx, res.ProviderID); err != nil {
			a.setError("auto-pull failed: " + err.Error())
		}
	}
}

// Check inspects the configured cloud target and returns what should happen.
// It records last_check_at in settings on every call, and updates
// pending_provider_id when Action="prompt" so the frontend can poll status
// and surface the conflict modal.
func (a *AutoSyncService) Check(ctx context.Context) SyncCheckResult {
	now := time.Now().UTC().Format(time.RFC3339)
	defer func() { _ = a.settings.Set("cloud.autosync.last_check_at", now) }()

	transportID, _ := a.settings.Get("cloud.autobackup.transport_id")
	if transportID == "" {
		// Fall back to autosync-specific override if set later, but for now
		// the auto-backup target IS the auto-sync source — single picker.
		return a.recordResult(SyncCheckResult{Action: "error", Message: "no cloud target configured"})
	}

	t, err := a.registry.Get(transportID)
	if err != nil {
		// Transport not yet registered (e.g. during startup, or disconnected).
		// Don't surface as an error — silently skip and try again next interval.
		return a.recordResult(SyncCheckResult{Action: "none", Message: "transport not available"})
	}

	listCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	blobs, err := t.List(listCtx)
	if err != nil {
		return a.recordResult(SyncCheckResult{Action: "error", Message: "list failed: " + err.Error()})
	}

	mostRecent, mostRecentTime := pickMostRecentBackup(blobs)
	if mostRecent == nil {
		_ = a.settings.Set("cloud.autosync.last_error", "")
		return a.recordResult(SyncCheckResult{Action: "none", Message: "no cloud backups found"})
	}

	lastSyncedAt := a.lastSyncedAt()
	// If the cloud blob was created at or before our last sync (we either
	// pulled it or uploaded around that time), nothing to do.
	if !mostRecentTime.After(lastSyncedAt) {
		_ = a.clearPending()
		_ = a.settings.Set("cloud.autosync.last_error", "")
		return a.recordResult(SyncCheckResult{
			Action:          "none",
			ProviderID:      mostRecent.ID,
			Filename:        mostRecent.Filename,
			CloudModifiedAt: mostRecentTime.UTC().Format(time.RFC3339),
			LastSyncedAt:    lastSyncedAt.UTC().Format(time.RFC3339),
		})
	}

	// Cloud is newer. Has the user already explicitly skipped this very blob?
	skipped, _ := a.settings.Get("cloud.autosync.last_skipped_provider_id")
	if skipped != "" && skipped == mostRecent.ID {
		_ = a.clearPending()
		_ = a.settings.Set("cloud.autosync.last_error", "")
		return a.recordResult(SyncCheckResult{
			Action:          "skipped",
			ProviderID:      mostRecent.ID,
			Filename:        mostRecent.Filename,
			CloudModifiedAt: mostRecentTime.UTC().Format(time.RFC3339),
		})
	}

	// Cloud newer + not skipped. Decide: auto-pull or prompt?
	localModifiedAt := a.localModifiedAt()
	localDirty := localModifiedAt.After(lastSyncedAt)

	if !localDirty {
		_ = a.clearPending()
		_ = a.settings.Set("cloud.autosync.last_error", "")
		return a.recordResult(SyncCheckResult{
			Action:          "auto_pull",
			ProviderID:      mostRecent.ID,
			Filename:        mostRecent.Filename,
			CloudModifiedAt: mostRecentTime.UTC().Format(time.RFC3339),
			LocalModifiedAt: localModifiedAt.UTC().Format(time.RFC3339),
			LastSyncedAt:    lastSyncedAt.UTC().Format(time.RFC3339),
		})
	}

	_ = a.settings.Set("cloud.autosync.pending_provider_id", mostRecent.ID)
	_ = a.settings.Set("cloud.autosync.pending_filename", mostRecent.Filename)
	_ = a.settings.Set("cloud.autosync.pending_cloud_modified_at", mostRecentTime.UTC().Format(time.RFC3339))
	_ = a.settings.Set("cloud.autosync.last_error", "")
	return a.recordResult(SyncCheckResult{
		Action:          "prompt",
		ProviderID:      mostRecent.ID,
		Filename:        mostRecent.Filename,
		CloudModifiedAt: mostRecentTime.UTC().Format(time.RFC3339),
		LocalModifiedAt: localModifiedAt.UTC().Format(time.RFC3339),
		LastSyncedAt:    lastSyncedAt.UTC().Format(time.RFC3339),
	})
}

// PullProviderID downloads + imports the given cloud blob and clears any
// pending-prompt state. Used by both the auto-pull goroutine path and the
// HTTP "user clicked Just import" path.
func (a *AutoSyncService) PullProviderID(ctx context.Context, providerID string) error {
	return a.pull(ctx, providerID)
}

func (a *AutoSyncService) pull(ctx context.Context, providerID string) error {
	if providerID == "" {
		return errors.New("provider_id required")
	}
	transportID, _ := a.settings.Get("cloud.autobackup.transport_id")
	if transportID == "" {
		return errors.New("no cloud target configured")
	}
	t, err := a.registry.Get(transportID)
	if err != nil {
		return errors.New("transport not available: " + err.Error())
	}

	dlCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	rc, err := t.Download(dlCtx, cloud.BlobRef{ID: providerID})
	if err != nil {
		return err
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, 100<<20))
	if err != nil {
		return err
	}
	if len(data) >= 100<<20 {
		return errors.New("backup file too large (>100 MB)")
	}

	opts := backup.ImportOptions{
		Mode: backup.ImportModeFullReplace,
	}
	// Auto-sync only ever consumes master-key encrypted blobs. Plain or
	// legacy-passphrase backups would require a UI prompt for a passphrase,
	// which we don't have in the goroutine path — fail clearly.
	if backup.IsEncrypted(data) {
		mode, mErr := backup.DetectEncryptMode(data)
		if mErr != nil {
			return mErr
		}
		if mode != backup.EncryptModeMaster {
			return errors.New("backup is not master-key encrypted; use Import panel manually")
		}
		seed, sErr := a.resolveSeed()
		if sErr != nil {
			return errors.New("keychain error: " + sErr.Error())
		}
		if seed == nil {
			return errors.New("master_key_not_configured: this backup requires the master recovery phrase")
		}
		opts.MasterSeed = seed
	}

	if _, err := a.importer.Import(bytes.NewReader(data), opts); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_ = a.settings.Set("cloud.autosync.last_pulled_at", now)
	_ = a.settings.Set("cloud.autosync.last_error", "")
	_ = a.clearPending()
	// A successful pull clears the skip-marker too — they referenced an old
	// blob, and we want fresh prompts next time.
	_ = a.settings.Set("cloud.autosync.last_skipped_provider_id", "")
	log.Printf("autosync: pulled %s", providerID)
	return nil
}

// SkipProviderID records the user's "Skip" decision so we don't re-prompt for
// the same blob. Cleared automatically when a newer blob shows up.
func (a *AutoSyncService) SkipProviderID(providerID string) {
	if providerID == "" {
		return
	}
	_ = a.settings.Set("cloud.autosync.last_skipped_provider_id", providerID)
	_ = a.clearPending()
}

func (a *AutoSyncService) clearPending() error {
	_ = a.settings.Set("cloud.autosync.pending_provider_id", "")
	_ = a.settings.Set("cloud.autosync.pending_filename", "")
	_ = a.settings.Set("cloud.autosync.pending_cloud_modified_at", "")
	return nil
}

func (a *AutoSyncService) lastSyncedAt() time.Time {
	pulledStr, _ := a.settings.Get("cloud.autosync.last_pulled_at")
	pulledAt, _ := time.Parse(time.RFC3339, pulledStr)
	pushedStr, _ := a.settings.Get("cloud.autobackup.last_run_at")
	pushedAt, _ := time.Parse(time.RFC3339, pushedStr)
	if pushedAt.After(pulledAt) {
		return pushedAt
	}
	return pulledAt
}

func (a *AutoSyncService) localModifiedAt() time.Time {
	v, _ := a.settings.Get("data.last_write_at")
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (a *AutoSyncService) resolveSeed() ([]byte, error) {
	if a.kc == nil {
		return nil, nil
	}
	_, seed, err := keychain.GetMasterKey(a.kc)
	if errors.Is(err, keychain.ErrNoMasterKey) {
		return nil, nil
	}
	return seed, err
}

func (a *AutoSyncService) setError(msg string) {
	log.Printf("autosync error: %s", msg)
	_ = a.settings.Set("cloud.autosync.last_error", msg)
}

func (a *AutoSyncService) recordResult(r SyncCheckResult) SyncCheckResult {
	a.mu.Lock()
	a.lastRun = r
	a.mu.Unlock()
	if r.Action == "error" && r.Message != "" {
		_ = a.settings.Set("cloud.autosync.last_error", r.Message)
	}
	return r
}

// LastResult returns the most recent Check result without re-running.
func (a *AutoSyncService) LastResult() SyncCheckResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastRun
}

// pickMostRecentBackup picks the BlobRef with the latest timestamp from a
// .tidybill backup list. Prefers ModifiedAt when present and non-zero;
// falls back to the timestamp embedded in the filename
// (tidybill-backup-2006-01-02T15-04-05Z.tidybill).
func pickMostRecentBackup(blobs []cloud.BlobRef) (*cloud.BlobRef, time.Time) {
	var best *cloud.BlobRef
	var bestTime time.Time
	for i := range blobs {
		b := &blobs[i]
		if !strings.HasSuffix(b.Filename, ".tidybill") {
			continue
		}
		t := blobTime(b)
		if t.IsZero() {
			continue
		}
		if best == nil || t.After(bestTime) {
			best = b
			bestTime = t
		}
	}
	return best, bestTime
}

// blobTime returns the most reliable timestamp for ordering. Some backends
// (notably Proton Drive via rclone) historically returned a zero ModTime;
// the filename always has it so that's our fallback.
func blobTime(b *cloud.BlobRef) time.Time {
	if !b.ModifiedAt.IsZero() {
		return b.ModifiedAt
	}
	return parseBackupFilenameTime(b.Filename)
}

// parseBackupFilenameTime extracts the ISO timestamp portion of a
// tidybill-backup-YYYY-MM-DDTHH-MM-SSZ.tidybill filename. Returns zero on
// any parse failure.
func parseBackupFilenameTime(filename string) time.Time {
	const prefix = "tidybill-backup-"
	const suffix = ".tidybill"
	if !strings.HasPrefix(filename, prefix) || !strings.HasSuffix(filename, suffix) {
		return time.Time{}
	}
	stamp := strings.TrimSuffix(strings.TrimPrefix(filename, prefix), suffix)
	t, err := time.Parse("2006-01-02T15-04-05Z", stamp)
	if err != nil {
		return time.Time{}
	}
	return t
}
