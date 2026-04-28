package localfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adamSHA256/tidybill/internal/cloud"
)

const TransportID = "local"

type Transport struct {
	dir string
}

func New(dir string) *Transport {
	return &Transport{dir: dir}
}

func (t *Transport) ID() string { return TransportID }

func (t *Transport) Upload(ctx context.Context, filename string, body io.Reader, size int64) (cloud.BlobRef, error) {
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return cloud.BlobRef{}, err
	}
	path := filepath.Join(t.dir, filename)

	tmp, err := os.CreateTemp(t.dir, ".upload-*.tmp")
	if err != nil {
		return cloud.BlobRef{}, err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	written, err := io.Copy(tmp, body)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return cloud.BlobRef{}, err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return cloud.BlobRef{}, err
	}

	info, statErr := os.Stat(path)
	var mtime time.Time
	if statErr == nil {
		mtime = info.ModTime()
	}

	// Detect encryption.
	enc := false
	if f, openErr := os.Open(path); openErr == nil {
		var head [6]byte
		n, _ := f.Read(head[:])
		_ = f.Close()
		enc = cloud.IsEncryptedPrefix(head[:n])
	}

	return cloud.BlobRef{
		ID:         path,
		Filename:   filename,
		Size:       written,
		ModifiedAt: mtime,
		Encrypted:  enc,
	}, nil
}

func (t *Transport) List(ctx context.Context) ([]cloud.BlobRef, error) {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var out []cloud.BlobRef
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "tidybill-backup-") || !strings.HasSuffix(name, ".tidybill") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(t.dir, name)

		enc := false
		if f, openErr := os.Open(path); openErr == nil {
			var head [6]byte
			n, _ := f.Read(head[:])
			_ = f.Close()
			enc = cloud.IsEncryptedPrefix(head[:n])
		}

		out = append(out, cloud.BlobRef{
			ID:         path,
			Filename:   name,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Encrypted:  enc,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModifiedAt.After(out[j].ModifiedAt) })
	return out, nil
}

func (t *Transport) Download(ctx context.Context, ref cloud.BlobRef) (io.ReadCloser, error) {
	// Defence: ref.ID is a full path; reject anything that escapes t.dir.
	abs, err := filepath.Abs(ref.ID)
	if err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(t.dir)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) && abs != rootAbs {
		return nil, fmt.Errorf("localfs: refusing to read outside of %s", rootAbs)
	}
	return os.Open(abs)
}

func (t *Transport) Delete(ctx context.Context, ref cloud.BlobRef) error {
	// Same escape-check as Download.
	abs, err := filepath.Abs(ref.ID)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(t.dir)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) && abs != rootAbs {
		return fmt.Errorf("localfs: refusing to delete outside of %s", rootAbs)
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (t *Transport) Status(ctx context.Context) (cloud.Status, error) {
	if _, err := os.Stat(t.dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return cloud.Status{Connected: false, Detail: err.Error()}, nil
	}
	return cloud.Status{Connected: true, AccountLabel: t.dir}, nil
}
