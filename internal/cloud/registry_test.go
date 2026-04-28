package cloud

import (
	"context"
	"errors"
	"io"
	"testing"
)

// fakeTransport is a no-op Transport implementation for testing.
type fakeTransport struct {
	id string
}

func (f *fakeTransport) ID() string { return f.id }

func (f *fakeTransport) Upload(_ context.Context, _ string, _ io.Reader, _ int64) (BlobRef, error) {
	return BlobRef{}, nil
}

func (f *fakeTransport) List(_ context.Context) ([]BlobRef, error) { return nil, nil }

func (f *fakeTransport) Download(_ context.Context, _ BlobRef) (io.ReadCloser, error) {
	return nil, nil
}

func (f *fakeTransport) Delete(_ context.Context, _ BlobRef) error { return nil }

func (f *fakeTransport) Status(_ context.Context) (Status, error) {
	return Status{Connected: true}, nil
}

func TestRegistryRegisterGetListUnregister(t *testing.T) {
	reg := NewRegistry()

	// Initially empty
	if ts := reg.List(); len(ts) != 0 {
		t.Fatalf("expected empty list, got %d", len(ts))
	}

	// Register two transports
	a := &fakeTransport{id: "alpha"}
	b := &fakeTransport{id: "beta"}
	reg.Register(a)
	reg.Register(b)

	// List returns both, sorted by ID
	ts := reg.List()
	if len(ts) != 2 {
		t.Fatalf("expected 2 transports, got %d", len(ts))
	}
	if ts[0].ID() != "alpha" || ts[1].ID() != "beta" {
		t.Fatalf("wrong order: %s %s", ts[0].ID(), ts[1].ID())
	}

	// Get returns correct transport
	got, err := reg.Get("alpha")
	if err != nil {
		t.Fatalf("Get alpha: %v", err)
	}
	if got.ID() != "alpha" {
		t.Fatalf("Get alpha returned %q", got.ID())
	}

	// Get unknown returns ErrTransportNotFound
	_, err = reg.Get("unknown")
	if !errors.Is(err, ErrTransportNotFound) {
		t.Fatalf("Get unknown: got %v, want ErrTransportNotFound", err)
	}

	// Unregister removes a transport
	reg.Unregister("alpha")
	_, err = reg.Get("alpha")
	if !errors.Is(err, ErrTransportNotFound) {
		t.Fatalf("Get after Unregister: got %v, want ErrTransportNotFound", err)
	}

	// List now has only beta
	ts = reg.List()
	if len(ts) != 1 || ts[0].ID() != "beta" {
		t.Fatalf("expected [beta], got %v", ts)
	}
}
