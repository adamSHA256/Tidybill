package cloud

import (
	"context"
	"io"
	"time"
)

// Transport abstracts a destination (local folder or cloud service)
// for .tidybill backup blobs.
type Transport interface {
	ID() string
	Upload(ctx context.Context, filename string, body io.Reader, size int64) (BlobRef, error)
	List(ctx context.Context) ([]BlobRef, error)
	Download(ctx context.Context, ref BlobRef) (io.ReadCloser, error)
	Delete(ctx context.Context, ref BlobRef) error
	Status(ctx context.Context) (Status, error)
}

type BlobRef struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Encrypted  bool      `json:"encrypted"`
}

type Status struct {
	Connected    bool   `json:"connected"`
	AccountLabel string `json:"account_label,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

// IsEncryptedPrefix returns true if the first six bytes of head match
// the TBILL\x01 magic from internal/backup.
func IsEncryptedPrefix(head []byte) bool {
	const magic = "TBILL\x01"
	return len(head) >= len(magic) && string(head[:len(magic)]) == magic
}
