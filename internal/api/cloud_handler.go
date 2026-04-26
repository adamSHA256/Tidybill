package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adamSHA256/tidybill/internal/backup"
	"github.com/adamSHA256/tidybill/internal/cloud"
	"github.com/adamSHA256/tidybill/internal/cloud/gdrive"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
	"github.com/adamSHA256/tidybill/internal/cloud/rclone"
	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database/repository"
)

// applyBackendExtras injects backend-specific parameters that the user never
// touches but rclone needs sent on every config/create. For Proton Drive this
// is mandatory: the default app_version (`macos-drive@1.0.0-alpha.1+rclone`)
// is fingerprinted by Proton's anti-abuse system and triggers Code=2028 at the
// SRP login endpoint. The `external-drive-*` prefix is the sanctioned
// third-party identifier per Proton engineer dlaumen
// (https://github.com/rclone/rclone/pull/9189). enable_caching=false +
// replace_existing_draft=true reduce concurrent metadata calls and let failed
// uploads retry cleanly. Called from both initial connect and reregister.
func applyBackendExtras(backendID string, params map[string]string) {
	if backendID != "protondrive" {
		return
	}
	// Identify ourselves truthfully — we are TidyBill, not rclone.
	// Per Proton SDK acceptable-use policy, the header must accurately
	// represent the application; spoofing is forbidden.
	params["app_version"] = "external-drive-tidybill@" + config.Version
	params["enable_caching"] = "false"
	params["replace_existing_draft"] = "true"
}

// pendingGDriveConnect holds the PKCE verifier for an in-flight OAuth flow.
type pendingGDriveConnect struct {
	Verifier string
}

// cloudTmpDir returns the directory used for transient files during
// cloud uploads/downloads. Created with MkdirAll on first use.
func (s *Server) cloudTmpDir() string {
	dir := filepath.Join(s.cfg.DataDir, "tmp")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// GET /api/cloud/transports
// Response: { transports: [{ id: string, status: cloud.Status }] }
func (s *Server) handleCloudTransports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transports := s.cloudRegistry.List()

	type transportInfo struct {
		ID     string       `json:"id"`
		Status cloud.Status `json:"status"`
	}
	out := make([]transportInfo, 0, len(transports))
	for _, t := range transports {
		st, _ := t.Status(ctx)
		out = append(out, transportInfo{ID: t.ID(), Status: st})
	}
	writeJSON(w, http.StatusOK, map[string]any{"transports": out})
}

// GET /api/cloud/rclone/backends
// Response: the rclone.Backends slice serialised as JSON.
func (s *Server) handleCloudRcloneBackends(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"backends": rclone.Backends})
}

// POST /api/cloud/gdrive/connect
func (s *Server) handleCloudGDriveConnect(w http.ResponseWriter, r *http.Request) {
	if gdrive.ClientID == "" {
		writeError(w, http.StatusInternalServerError, "OAuth client not configured")
		return
	}

	verifier, challenge := gdrive.GeneratePKCE()
	state := gdrive.RandomState()

	port, shutdown, resultChan, err := gdrive.StartLoopbackListener(state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start OAuth listener: "+err.Error())
		return
	}

	// Insert into pending-state map AFTER the listener is up. The completion
	// goroutine below always deletes its entry on exit (success, failure, or
	// 5-minute timeout), so once inserted the map self-cleans — no cap needed.
	s.gdriveConnectMu.Lock()
	s.gdriveConnectStates[state] = pendingGDriveConnect{Verifier: verifier}
	s.gdriveConnectMu.Unlock()

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", port)
	authURL := gdrive.BuildAuthURL(redirectURI, state, challenge)

	// Goroutine handles the OAuth result in the background.
	go func() {
		deadline := 5 * time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), deadline)
		defer cancel()
		defer shutdown()
		defer func() {
			s.gdriveConnectMu.Lock()
			delete(s.gdriveConnectStates, state)
			s.gdriveConnectMu.Unlock()
		}()

		var result gdrive.LoopbackResult
		select {
		case result = <-resultChan:
		case <-ctx.Done():
			s.recordGDriveError(ctx, "OAuth flow timed out")
			return
		}

		if result.Err != nil {
			s.recordGDriveError(ctx, result.Err.Error())
			return
		}

		// Look up verifier.
		s.gdriveConnectMu.Lock()
		pending, ok := s.gdriveConnectStates[result.State]
		s.gdriveConnectMu.Unlock()
		if !ok {
			s.recordGDriveError(ctx, "state not found")
			return
		}

		token, err := gdrive.ExchangeCode(ctx, result.Code, pending.Verifier, redirectURI)
		if err != nil {
			s.recordGDriveError(ctx, "token exchange failed: "+err.Error())
			return
		}

		email, _ := gdrive.FetchUserEmail(ctx, token)

		// Persist refresh token.
		if s.kc == nil {
			s.recordGDriveError(ctx, "keychain unavailable")
			return
		}
		if err := s.kc.Set(keychain.AcctGDriveRefreshToken, token.RefreshToken); err != nil {
			s.recordGDriveError(ctx, "keychain write failed: "+err.Error())
			return
		}

		// Construct and register transport.
		t, err := gdrive.New(s.kc, s.settings)
		if err != nil {
			s.recordGDriveError(ctx, "gdrive init: "+err.Error())
			return
		}
		s.cloudRegistry.Register(t)

		// Upsert cloud_configs.
		if s.cloudConfigs != nil {
			_ = s.cloudConfigs.Upsert(ctx, repository.CloudConfig{
				TransportID:  "gdrive",
				Enabled:      true,
				AccountLabel: sql.NullString{String: email, Valid: email != ""},
			})
		}
		log.Printf("gdrive: connected as %s", email)
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"auth_url": authURL,
		"state":    state,
	})
}

// recordGDriveError saves a last_error into cloud_configs so the frontend can surface it.
func (s *Server) recordGDriveError(ctx context.Context, msg string) {
	log.Printf("gdrive connect error: %s", msg)
	if s.cloudConfigs == nil {
		return
	}
	pc, _ := json.Marshal(map[string]string{"last_error": msg})
	_ = s.cloudConfigs.Upsert(ctx, repository.CloudConfig{
		TransportID:  "gdrive",
		Enabled:      false,
		PublicConfig: sql.NullString{String: string(pc), Valid: true},
	})
}

// POST /api/cloud/gdrive/disconnect
func (s *Server) handleCloudGDriveDisconnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.kc != nil {
		rt, err := s.kc.Get(keychain.AcctGDriveRefreshToken)
		if err == nil && rt != "" {
			_ = gdrive.RevokeToken(ctx, rt)
		}
		_ = s.kc.Delete(keychain.AcctGDriveRefreshToken)
	}

	if s.cloudConfigs != nil {
		_ = s.cloudConfigs.Disable(ctx, "gdrive")
	}
	s.cloudRegistry.Unregister("gdrive")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/cloud/rclone/{backend}/connect
func (s *Server) handleCloudRcloneConnect(w http.ResponseWriter, r *http.Request) {
	backendID := r.PathValue("backend")
	backend := rclone.FindBackend(backendID)
	if backend == nil {
		writeError(w, http.StatusNotFound, "unknown backend: "+backendID)
		return
	}

	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate required fields.
	var missing []string
	for _, f := range backend.Fields {
		if f.Generated {
			continue // generated fields are filled by rclone, not by the user
		}
		if f.Required && body[f.Name] == "" {
			missing = append(missing, f.Name)
		}
	}
	if len(missing) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"missing": missing})
		return
	}

	// Persist user-input fields to the keychain. Skip Transient (e.g. TOTP
	// codes that would be stale on next restart) and Generated (rclone fills
	// these after auth — captured below from config/dump).
	if s.kc != nil {
		for _, f := range backend.Fields {
			if f.Transient || f.Generated {
				continue
			}
			if v := body[f.Name]; v != "" {
				if err := s.kc.Set(keychain.RcloneAcct(backendID, f.Name), v); err != nil {
					writeError(w, http.StatusInternalServerError, "keychain write: "+err.Error())
					return
				}
			}
		}
	}

	// Ensure rcd is running.
	if s.rcloneMgr == nil {
		writeError(w, http.StatusInternalServerError, "rclone not available")
		return
	}
	ctx := r.Context()
	if err := s.rcloneMgr.EnsureRunning(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "rclone start: "+err.Error())
		return
	}

	addr, user, pass := s.rcloneMgr.Endpoint()
	rc := rclone.NewRC(addr, user, pass)

	remoteName := "tidybill-" + backend.ID
	parameters := map[string]string{}
	for _, f := range backend.Fields {
		if f.Generated {
			continue // not present in body on initial connect
		}
		if v := body[f.Name]; v != "" {
			parameters[f.Name] = v
		}
	}
	applyBackendExtras(backendID, parameters)
	if err := rc.Call(ctx, "config/create", map[string]any{
		"name":       remoteName,
		"type":       backend.Type,
		"parameters": parameters,
		"opt":        map[string]any{"obscure": true, "nonInteractive": true},
	}, nil); err != nil {
		s.cleanupFailedRcloneConnect(ctx, rc, backendID, remoteName, backend, body)
		if backendID == "protondrive" && !isRcloneTransportError(err) {
			status, code := mapProtonDriveError(err)
			writeJSON(w, status, map[string]any{"error_code": code, "error_raw": err.Error()})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Validate connectivity.
	if err := rc.Call(ctx, "operations/about", map[string]any{
		"fs": remoteName + ":",
	}, nil); err != nil {
		s.cleanupFailedRcloneConnect(ctx, rc, backendID, remoteName, backend, body)
		if backendID == "protondrive" && !isRcloneTransportError(err) {
			status, code := mapProtonDriveError(err)
			writeJSON(w, status, map[string]any{"error_code": code, "error_raw": err.Error()})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Capture rclone-generated session fields (e.g. Proton Drive's client_uid,
	// client_access_token, client_refresh_token, client_salted_key_pass) so we
	// can restore them on next app start without re-prompting the user for
	// credentials/2FA. Failure here is non-fatal: the connect succeeded, the
	// remote works, and at worst the user re-authenticates on next restart.
	if s.kc != nil {
		var dumped map[string]map[string]string
		if err := rc.Call(ctx, "config/dump", map[string]any{}, &dumped); err == nil {
			if remoteCfg, ok := dumped[remoteName]; ok {
				for _, f := range backend.Fields {
					if !f.Generated {
						continue
					}
					if v := remoteCfg[f.Name]; v != "" {
						if err := s.kc.Set(keychain.RcloneAcct(backendID, f.Name), v); err != nil {
							log.Printf("rclone connect %s: persist generated field %s: %v", backendID, f.Name, err)
						}
					}
				}
			} else {
				log.Printf("rclone connect %s: remote %q not in config/dump output", backendID, remoteName)
			}
		} else {
			log.Printf("rclone connect %s: config/dump failed: %v", backendID, err)
		}
	}

	// Determine bucketPath.
	bucketPath := "TidyBill"
	if backendID == "s3" {
		bucket := body["bucket"]
		if bucket != "" {
			bucketPath = bucket + "/TidyBill"
		}
	}

	// Build account label.
	accountLabel := buildRcloneAccountLabel(backendID, body)

	// Persist to cloud_configs.
	if s.cloudConfigs != nil {
		pc, _ := json.Marshal(map[string]string{
			"remote_name": remoteName,
			"bucket_path": bucketPath,
		})
		_ = s.cloudConfigs.Upsert(ctx, repository.CloudConfig{
			TransportID:  "rclone:" + backendID,
			Enabled:      true,
			AccountLabel: sql.NullString{String: accountLabel, Valid: true},
			PublicConfig: sql.NullString{String: string(pc), Valid: true},
		})
	}

	// Register transport.
	tr := rclone.New(s.rcloneMgr, backendID, remoteName, bucketPath, s.cloudTmpDir())
	s.cloudRegistry.Register(tr)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account_label": accountLabel})
}

func (s *Server) cleanupFailedRcloneConnect(ctx context.Context, rc *rclone.RC, backendID, remoteName string, backend *rclone.Backend, body map[string]string) {
	// Wipe keychain entries.
	if s.kc != nil {
		for _, f := range backend.Fields {
			_ = s.kc.Delete(keychain.RcloneAcct(backendID, f.Name))
		}
	}
	// Delete the partially-created rclone remote.
	_ = rc.Call(ctx, "config/delete", map[string]any{"name": remoteName}, nil)
}

func buildRcloneAccountLabel(backendID string, body map[string]string) string {
	switch backendID {
	case "sftp":
		return body["user"] + "@" + body["host"]
	case "webdav":
		return body["url"]
	case "s3":
		ep := body["endpoint"]
		if ep == "" {
			ep = body["region"]
		}
		return body["bucket"] + " @ " + ep
	case "protondrive":
		return body["username"]
	default:
		return backendID
	}
}

// isRcloneTransportError returns true for errors that indicate rclone's RC
// daemon wasn't reachable at all (not an auth failure from rclone itself).
// These should not be mapped to protondrive auth codes.
func isRcloneTransportError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "eof")
}

func mapProtonDriveError(rawErr error) (httpStatus int, code string) {
	msg := strings.ToLower(rawErr.Error())
	switch {
	case strings.Contains(msg, "incorrect login credentials"), strings.Contains(msg, "incorrect password"):
		return http.StatusUnauthorized, "invalid_credentials"
	case strings.Contains(msg, "two factor authentication required"), strings.Contains(msg, "2fa is required"):
		return http.StatusUnauthorized, "invalid_2fa"
	case strings.Contains(msg, "incorrect 2fa"), strings.Contains(msg, "invalid totp"):
		return http.StatusUnauthorized, "invalid_2fa"
	case strings.Contains(msg, "session expired"), strings.Contains(msg, "refresh token"):
		return http.StatusUnauthorized, "session_expired"
	case strings.Contains(msg, "unusual activity"), strings.Contains(msg, "code=2028"), strings.Contains(msg, "temporarily limited"):
		return http.StatusTooManyRequests, "rate_limited"
	default:
		return http.StatusInternalServerError, "generic"
	}
}

// POST /api/cloud/rclone/{backend}/disconnect
func (s *Server) handleCloudRcloneDisconnect(w http.ResponseWriter, r *http.Request) {
	backendID := r.PathValue("backend")
	ctx := r.Context()

	// Look up remote name from cloud_configs.public_config.
	var remoteName string
	if s.cloudConfigs != nil {
		rows, err := s.cloudConfigs.ListEnabled(ctx, "rclone:"+backendID)
		if err == nil && len(rows) > 0 && rows[0].PublicConfig.Valid {
			var pc struct {
				RemoteName string `json:"remote_name"`
			}
			if err := json.Unmarshal([]byte(rows[0].PublicConfig.String), &pc); err == nil {
				remoteName = pc.RemoteName
			}
		}
	}
	if remoteName == "" {
		remoteName = "tidybill-" + backendID
	}

	// Wipe keychain entries.
	backend := rclone.FindBackend(backendID)
	if backend != nil && s.kc != nil {
		for _, f := range backend.Fields {
			_ = s.kc.Delete(keychain.RcloneAcct(backendID, f.Name))
		}
	}

	if s.cloudConfigs != nil {
		_ = s.cloudConfigs.Disable(ctx, "rclone:"+backendID)
	}
	s.cloudRegistry.Unregister("rclone:" + backendID)

	// Delete rclone remote (best-effort).
	if s.rcloneMgr != nil {
		addr, user, pass := s.rcloneMgr.Endpoint()
		rc := rclone.NewRC(addr, user, pass)
		_ = rc.Call(ctx, "config/delete", map[string]any{"name": remoteName}, nil)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/cloud/upload
func (s *Server) handleCloudUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		TransportID string                `json:"transport_id"`
		Passphrase  string                `json:"passphrase"`
		Filters     *backup.ExportFilters `json:"filters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	t, err := s.cloudRegistry.Get(req.TransportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "transport not found: "+req.TransportID)
		return
	}

	var data []byte
	if req.Passphrase != "" {
		data, err = s.backupExport.ExportEncryptedJSON(req.Filters, req.Passphrase)
	} else {
		data, err = s.backupExport.ExportJSON(req.Filters)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "export failed: "+err.Error())
		return
	}

	// Filename uses ISO-8601 UTC down to seconds so two uploads in the
	// same day (or after a failed/canceled upload that left a zombie file
	// on the remote) don't collide. Proton Drive in particular returns
	// Code=2500 ("name already exists") on collision and the protondrive
	// backend's draft-replace fallback only works for true draft state,
	// not finalized or zombie files — so unique names are the only
	// reliable way to avoid the collision.
	filename := "tidybill-backup-" + time.Now().UTC().Format("2006-01-02T15-04-05Z") + ".tidybill"

	ref, err := t.Upload(ctx, filename, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "upload failed: "+err.Error())
		return
	}

	// Record upload in cloud_uploads.
	if s.cloudConfigs != nil {
		_ = s.cloudConfigs.InsertUpload(ctx, req.TransportID, filename, ref.ID, len(data), req.Passphrase != "")
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "blob_ref": ref})
}

// GET /api/cloud/{transport_id}/list
func (s *Server) handleCloudList(w http.ResponseWriter, r *http.Request) {
	transportID := r.PathValue("transport_id")
	t, err := s.cloudRegistry.Get(transportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "transport not found: "+transportID)
		return
	}

	blobs, err := t.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if blobs == nil {
		blobs = []cloud.BlobRef{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"blobs": blobs})
}

// POST /api/cloud/{transport_id}/download-preview
func (s *Server) handleCloudDownloadPreview(w http.ResponseWriter, r *http.Request) {
	transportID := r.PathValue("transport_id")
	t, err := s.cloudRegistry.Get(transportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "transport not found: "+transportID)
		return
	}

	var req struct {
		ProviderID  string `json:"provider_id"`
		Passphrase  string `json:"passphrase"`
		PreviewMode string `json:"preview_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ProviderID == "" {
		writeError(w, http.StatusBadRequest, "provider_id required")
		return
	}
	if req.PreviewMode == "" {
		req.PreviewMode = "merge"
	}

	ctx := r.Context()
	rc, err := t.Download(ctx, cloud.BlobRef{ID: req.ProviderID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "download failed: "+err.Error())
		return
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, 100<<20))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed: "+err.Error())
		return
	}
	if len(data) >= 100<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "backup file too large (>100 MB)")
		return
	}

	if backup.IsEncrypted(data) && req.Passphrase == "" {
		writeError(w, http.StatusBadRequest, "passphrase required")
		return
	}

	report, err := s.backupImport.Import(bytes.NewReader(data), backup.ImportOptions{
		Mode:        backup.ImportModePreview,
		Passphrase:  req.Passphrase,
		PreviewMode: req.PreviewMode,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// POST /api/cloud/{transport_id}/download-apply
func (s *Server) handleCloudDownloadApply(w http.ResponseWriter, r *http.Request) {
	transportID := r.PathValue("transport_id")
	t, err := s.cloudRegistry.Get(transportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "transport not found: "+transportID)
		return
	}

	var req struct {
		ProviderID string `json:"provider_id"`
		Passphrase string `json:"passphrase"`
		Mode       string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ProviderID == "" {
		writeError(w, http.StatusBadRequest, "provider_id required")
		return
	}

	// Map mode string to internal constant (same as handleImport).
	var mode string
	switch req.Mode {
	case "merge", "":
		mode = backup.ImportModeSmartMerge
	case "replace":
		mode = backup.ImportModeFullReplace
	case "force":
		mode = backup.ImportModeForce
	default:
		writeError(w, http.StatusBadRequest, "invalid mode: use merge, replace, or force")
		return
	}

	ctx := r.Context()
	rc, err := t.Download(ctx, cloud.BlobRef{ID: req.ProviderID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "download failed: "+err.Error())
		return
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, 100<<20))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed: "+err.Error())
		return
	}
	if len(data) >= 100<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "backup file too large (>100 MB)")
		return
	}

	if backup.IsEncrypted(data) && req.Passphrase == "" {
		writeError(w, http.StatusBadRequest, "passphrase required")
		return
	}

	report, err := s.backupImport.Import(bytes.NewReader(data), backup.ImportOptions{
		Mode:       mode,
		Passphrase: req.Passphrase,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// DELETE /api/cloud/{transport_id}/blob?provider_id=...
func (s *Server) handleCloudDeleteBlob(w http.ResponseWriter, r *http.Request) {
	transportID := r.PathValue("transport_id")
	providerID := r.URL.Query().Get("provider_id")
	if providerID == "" {
		writeError(w, http.StatusBadRequest, "provider_id query param required")
		return
	}

	t, err := s.cloudRegistry.Get(transportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "transport not found: "+transportID)
		return
	}

	if err := t.Delete(r.Context(), cloud.BlobRef{ID: providerID}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// reregisterRcloneRemotes restores previously-saved rclone remotes
// from cloud_configs + the keychain at process startup. Errors for any
// single remote are logged and skipped — a corrupted row must never
// prevent the app from booting.
func (s *Server) reregisterRcloneRemotes(ctx context.Context) {
	if s.kc == nil || s.rcloneMgr == nil {
		return
	}
	rows, err := s.cloudConfigs.ListEnabled(ctx, "rclone:")
	if err != nil {
		log.Printf("reregister rclone: list: %v", err)
		return
	}
	if len(rows) == 0 {
		return
	}
	if err := s.rcloneMgr.EnsureRunning(ctx); err != nil {
		log.Printf("reregister rclone: rcd not running: %v", err)
		return
	}
	addr, user, pass := s.rcloneMgr.Endpoint()
	rc := rclone.NewRC(addr, user, pass)

	for _, row := range rows {
		backendID := strings.TrimPrefix(row.TransportID, "rclone:")
		backend := rclone.FindBackend(backendID)
		if backend == nil {
			log.Printf("reregister rclone: unknown backend %q", backendID)
			continue
		}
		var pc struct {
			RemoteName string `json:"remote_name"`
			BucketPath string `json:"bucket_path"`
		}
		if row.PublicConfig.Valid {
			if err := json.Unmarshal([]byte(row.PublicConfig.String), &pc); err != nil {
				log.Printf("reregister rclone %s: parse public_config: %v", row.TransportID, err)
				continue
			}
		}
		if pc.RemoteName == "" {
			pc.RemoteName = "tidybill-" + backendID
		}

		// Restore ALL fields (user-input + Generated) from the keychain.
		// Transient fields (e.g. 2FA codes) were never persisted, so they're
		// naturally absent — that's what we want; rclone uses the saved
		// Generated session tokens to skip re-auth.
		params := map[string]string{}
		for _, f := range backend.Fields {
			v, err := s.kc.Get(keychain.RcloneAcct(backendID, f.Name))
			if err == nil && v != "" {
				params[f.Name] = v
			}
		}
		applyBackendExtras(backendID, params)
		if err := rc.Call(ctx, "config/create", map[string]any{
			"name":       pc.RemoteName,
			"type":       backend.Type,
			"parameters": params,
			"opt":        map[string]any{"obscure": true, "nonInteractive": true},
		}, nil); err != nil {
			log.Printf("reregister rclone %s: config/create: %v", row.TransportID, err)
			continue
		}

		s.cloudRegistry.Register(rclone.New(
			s.rcloneMgr, backendID, pc.RemoteName, pc.BucketPath, s.cloudTmpDir(),
		))
		log.Printf("reregister rclone %s ok", row.TransportID)
	}
}

// ShutdownCloud stops the rclone manager if running.
func (s *Server) ShutdownCloud() {
	if s.rcloneMgr != nil {
		_ = s.rcloneMgr.Stop()
	}
}

