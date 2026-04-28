package cloud

import (
	"errors"
	"sort"
	"sync"
)

var ErrTransportNotFound = errors.New("cloud: transport not registered")

type Registry struct {
	mu         sync.RWMutex
	transports map[string]Transport
}

func NewRegistry() *Registry {
	return &Registry{transports: make(map[string]Transport)}
}

func (r *Registry) Register(t Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transports[t.ID()] = t
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.transports, id)
}

func (r *Registry) Get(id string) (Transport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.transports[id]
	if !ok {
		return nil, ErrTransportNotFound
	}
	return t, nil
}

func (r *Registry) List() []Transport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Transport, 0, len(r.transports))
	for _, t := range r.transports {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}
