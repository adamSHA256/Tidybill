package gdrive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/oauth2"
	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/adamSHA256/tidybill/internal/cloud"
	"github.com/adamSHA256/tidybill/internal/cloud/keychain"
	"github.com/adamSHA256/tidybill/internal/database/repository"
)

var ErrNotConnected = errors.New("gdrive: not connected")

type Transport struct {
	kc       keychain.Store
	settings *repository.SettingsRepository

	mu       sync.Mutex
	email    string
	folderID string
	src      oauth2.TokenSource
	svc      *driveapi.Service
}

func New(kc keychain.Store, settings *repository.SettingsRepository) (*Transport, error) {
	rt, err := kc.Get(keychain.AcctGDriveRefreshToken)
	if err != nil {
		return nil, ErrNotConnected
	}
	if ClientID == "" {
		return nil, errors.New("gdrive: ClientID not injected at build time")
	}
	cfg := oauthConfig("") // redirect URL unused for refresh
	token := &oauth2.Token{RefreshToken: rt, Expiry: time.Now()}
	src := cfg.TokenSource(context.Background(), token)

	t := &Transport{kc: kc, settings: settings, src: src}
	return t, nil
}

func (t *Transport) ID() string { return TransportID }

func (t *Transport) ensureService(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.svc != nil {
		return nil
	}
	svc, err := driveapi.NewService(ctx, option.WithTokenSource(t.src))
	if err != nil {
		return err
	}
	t.svc = svc
	return nil
}

func (t *Transport) ensureFolder(ctx context.Context) (string, error) {
	if id, err := t.settings.Get("cloud.gdrive.folder_id"); err == nil && id != "" {
		// Verify still exists (non-trashed).
		if _, err := t.svc.Files.Get(id).Fields("id, trashed").Context(ctx).Do(); err == nil {
			t.mu.Lock()
			t.folderID = id
			t.mu.Unlock()
			return id, nil
		}
	}
	// Find or create.
	q := "name = '" + FolderName + "' and mimeType = '" + FolderMime + "' and trashed = false"
	list, err := t.svc.Files.List().Q(q).Spaces("drive").Fields("files(id)").Context(ctx).Do()
	if err != nil {
		return "", err
	}
	var id string
	if len(list.Files) > 0 {
		id = list.Files[0].Id
	} else {
		f, err := t.svc.Files.Create(&driveapi.File{Name: FolderName, MimeType: FolderMime}).Fields("id").Context(ctx).Do()
		if err != nil {
			return "", err
		}
		id = f.Id
	}
	_ = t.settings.Set("cloud.gdrive.folder_id", id)
	t.mu.Lock()
	t.folderID = id
	t.mu.Unlock()
	return id, nil
}

func (t *Transport) Upload(ctx context.Context, filename string, body io.Reader, size int64) (cloud.BlobRef, error) {
	if err := t.ensureService(ctx); err != nil {
		return cloud.BlobRef{}, err
	}
	folderID, err := t.ensureFolder(ctx)
	if err != nil {
		return cloud.BlobRef{}, err
	}
	// ResumableMedia requires io.ReaderAt (not io.Reader). Backup blobs
	// are small (a few hundred KB to a few MB — see PLAN §Non-goals on
	// differential transfer) so buffering in memory is fine for v1. If
	// blobs ever grow past ~20 MB this should switch to a tempfile.
	buf, err := io.ReadAll(body)
	if err != nil {
		return cloud.BlobRef{}, err
	}
	if int64(len(buf)) != size && size > 0 {
		return cloud.BlobRef{}, fmt.Errorf("gdrive: body length %d != declared size %d", len(buf), size)
	}
	encrypted := cloud.IsEncryptedPrefix(buf) // inspect content, not filename

	f := &driveapi.File{Name: filename, Parents: []string{folderID}}
	created, err := t.svc.Files.Create(f).
		ResumableMedia(ctx, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream").
		Fields("id, name, size, modifiedTime").
		Context(ctx).Do()
	if err != nil {
		return cloud.BlobRef{}, err
	}
	mt, _ := time.Parse(time.RFC3339, created.ModifiedTime)
	return cloud.BlobRef{
		ID:         created.Id,
		Filename:   created.Name,
		Size:       created.Size,
		ModifiedAt: mt,
		Encrypted:  encrypted,
	}, nil
}

func (t *Transport) List(ctx context.Context) ([]cloud.BlobRef, error) {
	if err := t.ensureService(ctx); err != nil {
		return nil, err
	}
	folderID, err := t.ensureFolder(ctx)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("'%s' in parents and trashed = false and name contains '.tidybill'", folderID)
	list, err := t.svc.Files.List().
		Q(q).
		Fields("files(id, name, size, modifiedTime)").
		OrderBy("modifiedTime desc").
		Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make([]cloud.BlobRef, 0, len(list.Files))
	for _, f := range list.Files {
		mt, _ := time.Parse(time.RFC3339, f.ModifiedTime)
		// Listing Drive does not give us the first 6 bytes; we cannot
		// cheaply determine encryption without a HEAD+range request per
		// file. Leave Encrypted false here; the download-preview step
		// inspects the magic bytes before handing to Import, which is
		// when it actually matters. UI can display "unknown" when List
		// returns Encrypted: false — or treat it as "assume encrypted"
		// and always prompt for passphrase.
		out = append(out, cloud.BlobRef{
			ID: f.Id, Filename: f.Name, Size: f.Size, ModifiedAt: mt,
			Encrypted: false,
		})
	}
	return out, nil
}

func (t *Transport) Download(ctx context.Context, ref cloud.BlobRef) (io.ReadCloser, error) {
	if err := t.ensureService(ctx); err != nil {
		return nil, err
	}
	resp, err := t.svc.Files.Get(ref.ID).Context(ctx).Download()
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (t *Transport) Delete(ctx context.Context, ref cloud.BlobRef) error {
	if err := t.ensureService(ctx); err != nil {
		return err
	}
	return t.svc.Files.Delete(ref.ID).Context(ctx).Do()
}

func (t *Transport) Status(ctx context.Context) (cloud.Status, error) {
	if err := t.ensureService(ctx); err != nil {
		return cloud.Status{Connected: false, Detail: err.Error()}, nil
	}
	// Best-effort user email (cached).
	t.mu.Lock()
	email := t.email
	t.mu.Unlock()
	if email == "" {
		if tok, err := t.src.Token(); err == nil {
			if e, err := FetchUserEmail(ctx, tok); err == nil {
				t.mu.Lock()
				t.email = e
				t.mu.Unlock()
				email = e
			}
		}
	}
	return cloud.Status{Connected: true, AccountLabel: email}, nil
}
