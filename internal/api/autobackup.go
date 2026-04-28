package api

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cloud"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
	"github.com/adamSHA256/tidybill/internal/database/repository"
)

// AutoBackupService runs a background goroutine that uploads a backup to the
// configured cloud transport after the user has been idle for at least
// idle_minutes following a DB write. It never blocks or errors DB write paths.
type AutoBackupService struct {
	settings *repository.SettingsRepository
	registry *cloud.Registry
	export   *backup.ExportService
	kc       keychain.Store
	trigger  chan struct{} // buffered(1): signals an immediate backup
	ctx      context.Context
	cancel   context.CancelFunc
}

func newAutoBackupService(
	settings *repository.SettingsRepository,
	registry *cloud.Registry,
	export *backup.ExportService,
	kc keychain.Store,
) *AutoBackupService {
	ctx, cancel := context.WithCancel(context.Background())
	a := &AutoBackupService{
		settings: settings,
		registry: registry,
		export:   export,
		kc:       kc,
		trigger:  make(chan struct{}, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
	// Clear any stale in-progress flag left by a previous crash.
	_ = settings.Set("cloud.autobackup.in_progress", "0")
	return a
}

// Shutdown cancels the goroutine and any in-progress upload.
// Safe to call multiple times.
func (a *AutoBackupService) Shutdown() {
	a.cancel()
}

// TriggerNow enqueues an immediate backup, bypassing the idle window.
// Non-blocking: if a trigger is already queued it is dropped (one is enough).
func (a *AutoBackupService) TriggerNow() {
	select {
	case a.trigger <- struct{}{}:
	default:
	}
}

// Run loops every 30 seconds and calls tick(). Exits when ctx is cancelled.
func (a *AutoBackupService) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.trigger:
			a.tick(true)
		case <-ticker.C:
			a.tick(false)
		}
	}
}

// tick checks whether an auto-backup should run and, if so, executes it.
// forceNow=true bypasses the idle-window check (manual trigger).
func (a *AutoBackupService) tick(forceNow bool) {
	enabled, _ := a.settings.Get("cloud.autobackup.enabled")
	if enabled != "1" && !forceNow {
		return
	}

	transportID, _ := a.settings.Get("cloud.autobackup.transport_id")
	if transportID == "" {
		return
	}

	t, err := a.registry.Get(transportID)
	if err != nil {
		return // transport disconnected — will retry naturally once reconnected
	}

	if !forceNow && !a.idleWindowPassed() {
		return
	}

	a.runBackup(t, transportID)
}

// idleWindowPassed returns true when:
//  1. At least one DB write has occurred (data.last_write_at is set)
//  2. idle_minutes have elapsed since that last write (user is idle)
//  3. The last write happened after the last successful backup (new data exists)
func (a *AutoBackupService) idleWindowPassed() bool {
	lastWriteStr, _ := a.settings.Get("data.last_write_at")
	if lastWriteStr == "" {
		return false // no writes ever recorded
	}
	lastWrite, err := time.Parse("2006-01-02T15:04:05Z", lastWriteStr)
	if err != nil {
		return false
	}

	idleMinStr, _ := a.settings.Get("cloud.autobackup.idle_minutes")
	idleMin, _ := strconv.Atoi(idleMinStr)
	if idleMin <= 0 {
		idleMin = 5
	}
	if time.Since(lastWrite) < time.Duration(idleMin)*time.Minute {
		return false // user still active
	}

	// If we've never backed up, proceed.
	lastRunStr, _ := a.settings.Get("cloud.autobackup.last_run_at")
	if lastRunStr == "" {
		return true
	}
	lastRun, err := time.Parse(time.RFC3339, lastRunStr)
	if err != nil {
		return true // unparseable — treat as stale
	}
	return lastWrite.After(lastRun)
}

// runBackup performs export + upload and records the result in settings.
// The upload context is derived from a.ctx, so closing the app cancels it cleanly.
func (a *AutoBackupService) runBackup(t cloud.Transport, transportID string) {
	log.Printf("autobackup: starting backup to %s", transportID)
	_ = a.settings.Set("cloud.autobackup.in_progress", "1")
	defer func() { _ = a.settings.Set("cloud.autobackup.in_progress", "0") }()

	seed, err := a.resolveSeed()
	if err != nil {
		a.setError("keychain error: " + err.Error())
		return
	}
	if seed == nil {
		a.setError("master key not configured — enable encryption first")
		return
	}

	data, err := a.export.ExportMasterEncryptedJSON(nil, seed)
	if err != nil {
		a.setError("export failed: " + err.Error())
		return
	}

	filename := "tidybill-backup-" + time.Now().UTC().Format("2006-01-02T15-04-05Z") + ".tidybill"

	// Upload context: inherit cancellation from a.ctx (shutdown signal) but
	// also cap at 5 minutes in case the remote hangs. If the app closes,
	// a.ctx is cancelled, which propagates here and aborts the upload cleanly
	// instead of leaving the goroutine blocked until the OS kills the process.
	uploadCtx, uploadCancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer uploadCancel()

	_, err = t.Upload(uploadCtx, filename, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("autobackup: upload cancelled (app shutting down)")
			return // don't record as an error — will retry next launch
		}
		a.setError("upload failed: " + err.Error())
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_ = a.settings.Set("cloud.autobackup.last_run_at", now)
	_ = a.settings.Set("cloud.autobackup.last_error", "")
	log.Printf("autobackup: backup complete → %s", filename)

	// Best-effort retention pass. A failure here doesn't undo the upload —
	// the user still has their fresh backup, just with extra older files.
	if err := a.pruneBackups(a.ctx, t); err != nil {
		log.Printf("autobackup-prune: %v", err)
	}
}

func (a *AutoBackupService) resolveSeed() ([]byte, error) {
	if a.kc == nil {
		return nil, nil
	}
	_, seed, err := keychain.GetMasterKey(a.kc)
	if errors.Is(err, keychain.ErrNoMasterKey) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return seed, nil
}

func (a *AutoBackupService) setError(msg string) {
	log.Printf("autobackup error: %s", msg)
	_ = a.settings.Set("cloud.autobackup.last_error", msg)
}
