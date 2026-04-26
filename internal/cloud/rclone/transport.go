package rclone

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

type Transport struct {
	mgr        *Manager
	backendID  string // "sftp", "webdav", ...
	remoteName string // the name registered with config/create (e.g. "tidybill-sftp")
	bucketPath string // optional prefix; e.g. "TidyBill" or "bucketname/TidyBill"
	tmpDir     string
}

// rcListItem is the JSON shape rclone's operations/list returns for
// each entry. Field names are CAPITALISED exactly as rclone emits them
// — do not change to lowercase or the decoder returns zero values.
type rcListItem struct {
	Path    string    `json:"Path"`
	Name    string    `json:"Name"`
	Size    int64     `json:"Size"`
	ModTime time.Time `json:"ModTime"`
	IsDir   bool      `json:"IsDir"`
}

// rcListResponse wraps the items.
type rcListResponse struct {
	List []rcListItem `json:"list"`
}

func New(mgr *Manager, backendID, remoteName, bucketPath, tmpDir string) *Transport {
	return &Transport{
		mgr: mgr, backendID: backendID, remoteName: remoteName,
		bucketPath: bucketPath, tmpDir: tmpDir,
	}
}

func (t *Transport) ID() string { return "rclone:" + t.backendID }

func (t *Transport) rc(ctx context.Context) (*RC, error) {
	if err := t.mgr.EnsureRunning(ctx); err != nil {
		return nil, err
	}
	addr, user, pass := t.mgr.Endpoint()
	return NewRC(addr, user, pass), nil
}

func (t *Transport) fs() string {
	if t.bucketPath == "" {
		return t.remoteName + ":"
	}
	return t.remoteName + ":" + t.bucketPath
}

func (t *Transport) Upload(ctx context.Context, filename string, body io.Reader, size int64) (cloud.BlobRef, error) {
	// Tee body to detect encryption prefix before writing to tempfile.
	var headBuf bytes.Buffer
	teeReader := io.TeeReader(io.LimitReader(body, 6), &headBuf)
	// Read the first 6 bytes via tee, then concatenate remainder.
	head := make([]byte, 6)
	n, _ := teeReader.Read(head)
	encrypted := cloud.IsEncryptedPrefix(head[:n])

	// Write tempfile: header bytes + rest of body.
	if err := os.MkdirAll(t.tmpDir, 0o700); err != nil {
		return cloud.BlobRef{}, err
	}
	tmpPath := filepath.Join(t.tmpDir, "upload-"+randomHex(8)+".tidybill")
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return cloud.BlobRef{}, err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := f.Write(head[:n]); err != nil {
		f.Close()
		return cloud.BlobRef{}, err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return cloud.BlobRef{}, err
	}
	if err := f.Close(); err != nil {
		return cloud.BlobRef{}, err
	}

	absPath, err := filepath.Abs(tmpPath)
	if err != nil {
		return cloud.BlobRef{}, err
	}

	rc, err := t.rc(ctx)
	if err != nil {
		return cloud.BlobRef{}, err
	}

	// Ensure destination directory exists (idempotent).
	if err := rc.Call(ctx, "operations/mkdir", map[string]any{
		"fs":     t.remoteName + ":",
		"remote": t.bucketPath,
	}, nil); err != nil {
		return cloud.BlobRef{}, fmt.Errorf("mkdir: %w", err)
	}

	// Copy from local tempfile to remote.
	// IMPORTANT: rclone's `:local:` connection string defaults its root to
	// rcd's CWD, so passing an absolute path in `srcRemote` makes rclone
	// look for `<rcd-cwd>/<absolute-path>` and fail with 404 "object not
	// found". Use the split form: put the directory in srcFs (so it
	// becomes the Fs root), and the bare filename in srcRemote.
	// Same fix on Download below. See rclone forum thread 15608.
	if err := rc.Call(ctx, "operations/copyfile", map[string]any{
		"srcFs":     ":local:" + filepath.Dir(absPath),
		"srcRemote": filepath.Base(absPath),
		"dstFs":     t.fs(),
		"dstRemote": filename,
	}, nil); err != nil {
		return cloud.BlobRef{}, fmt.Errorf("copyfile: %w", err)
	}

	// Stat the uploaded file.
	var statResult struct {
		Size    int64     `json:"Size"`
		ModTime time.Time `json:"ModTime"`
	}
	if err := rc.Call(ctx, "operations/stat", map[string]any{
		"fs":     t.fs(),
		"remote": filename,
	}, &statResult); err != nil {
		// Non-fatal: return a best-effort ref.
		return cloud.BlobRef{
			ID:        t.bucketPath + "/" + filename,
			Filename:  filename,
			Encrypted: encrypted,
		}, nil
	}

	return cloud.BlobRef{
		ID:         t.bucketPath + "/" + filename,
		Filename:   filename,
		Size:       statResult.Size,
		ModifiedAt: statResult.ModTime,
		Encrypted:  encrypted,
	}, nil
}

func (t *Transport) List(ctx context.Context) ([]cloud.BlobRef, error) {
	rc, err := t.rc(ctx)
	if err != nil {
		return nil, err
	}

	var resp rcListResponse
	if err := rc.Call(ctx, "operations/list", map[string]any{
		"fs":     t.fs(),
		"remote": "",
	}, &resp); err != nil {
		return nil, err
	}

	var out []cloud.BlobRef
	for _, item := range resp.List {
		if item.IsDir {
			continue
		}
		if !strings.HasSuffix(item.Name, ".tidybill") {
			continue
		}
		// BlobRef.ID is the full remote path so the delete handler can find it.
		id := item.Path
		if id == "" {
			id = t.bucketPath + "/" + item.Name
		}
		out = append(out, cloud.BlobRef{
			ID:         id,
			Filename:   item.Name,
			Size:       item.Size,
			ModifiedAt: item.ModTime,
			// Encrypted is left false — we cannot cheaply inspect the
			// first 6 bytes without a separate download per file.
			Encrypted: false,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModifiedAt.After(out[j].ModifiedAt) })
	return out, nil
}

func (t *Transport) Download(ctx context.Context, ref cloud.BlobRef) (io.ReadCloser, error) {
	rc, err := t.rc(ctx)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(t.tmpDir, 0o700); err != nil {
		return nil, err
	}
	tmpPath := filepath.Join(t.tmpDir, "download-"+randomHex(8)+".tidybill")
	absTmpPath, err := filepath.Abs(tmpPath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(ref.ID)
	// See Upload() comment about :local: + absolute path → 404. Split here too.
	if err := rc.Call(ctx, "operations/copyfile", map[string]any{
		"srcFs":     t.fs(),
		"srcRemote": filename,
		"dstFs":     ":local:" + filepath.Dir(absTmpPath),
		"dstRemote": filepath.Base(absTmpPath),
	}, nil); err != nil {
		return nil, fmt.Errorf("download copyfile: %w", err)
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	return &cleanupReader{f: f, path: tmpPath}, nil
}

func (t *Transport) Delete(ctx context.Context, ref cloud.BlobRef) error {
	rc, err := t.rc(ctx)
	if err != nil {
		return err
	}
	filename := filepath.Base(ref.ID)
	return rc.Call(ctx, "operations/deletefile", map[string]any{
		"fs":     t.fs(),
		"remote": filename,
	}, nil)
}

func (t *Transport) Status(ctx context.Context) (cloud.Status, error) {
	rc, err := t.rc(ctx)
	if err != nil {
		return cloud.Status{Connected: false, Detail: err.Error()}, nil
	}
	var about struct {
		Total int64 `json:"total"`
	}
	if err := rc.Call(ctx, "operations/about", map[string]any{
		"fs": t.remoteName + ":",
	}, &about); err != nil {
		return cloud.Status{Connected: false, Detail: err.Error()}, nil
	}
	return cloud.Status{Connected: true, AccountLabel: t.bucketPath}, nil
}

// cleanupReader deletes its tempfile on Close.
type cleanupReader struct {
	f    *os.File
	path string
	once sync.Once
}

func (c *cleanupReader) Read(p []byte) (int, error) { return c.f.Read(p) }
func (c *cleanupReader) Close() error {
	var err error
	c.once.Do(func() {
		err = c.f.Close()
		_ = os.Remove(c.path)
	})
	return err
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// rcStatResult is used in Upload for stat response deserialization.
// Defined separately to make the JSON field mapping explicit.
type rcStatResult struct {
	Item struct {
		Size    int64  `json:"Size"`
		ModTime string `json:"ModTime"`
	} `json:"item"`
}

// marshalPublicConfig returns the JSON for cloud_configs.public_config.
func marshalPublicConfig(remoteName, bucketPath string) (string, error) {
	b, err := json.Marshal(map[string]string{
		"remote_name": remoteName,
		"bucket_path": bucketPath,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
