package main

import (
	"context"
	"sync"
)

// Link is the metadata row. The file itself lives in the object store;
// PreviewKey is the only thing that connects the two.
type Link struct {
	Code       string
	TargetURL  string
	Title      string
	PreviewKey string
}

type Repo interface {
	Create(ctx context.Context, l Link) error
}

// MemRepo stands in for the SQL table. Err lets a test make the insert fail.
type MemRepo struct {
	mu    sync.Mutex
	Err   error
	links []Link
}

func (m *MemRepo) Create(_ context.Context, l Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.links = append(m.links, l)
	return nil
}

func (m *MemRepo) All() []Link {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Link(nil), m.links...)
}
