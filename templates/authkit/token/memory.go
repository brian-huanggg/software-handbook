package token

import (
	"context"
	"sync"
	"time"
)

type memoryRefreshStore struct {
	mu sync.Mutex
	m  map[string]Refresh // key: string(IDHash)
}

// NewMemoryRefreshStore returns an in-process RefreshStore for tests and dev.
func NewMemoryRefreshStore() RefreshStore {
	return &memoryRefreshStore{m: make(map[string]Refresh)}
}

func cloneRefresh(r Refresh) Refresh {
	r.IDHash = append([]byte(nil), r.IDHash...)
	if r.UsedAt != nil {
		t := *r.UsedAt
		r.UsedAt = &t
	}
	return r
}

func (s *memoryRefreshStore) Create(_ context.Context, r Refresh) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[string(r.IDHash)]; ok {
		return ErrDuplicate
	}
	s.m[string(r.IDHash)] = cloneRefresh(r)
	return nil
}

func (s *memoryRefreshStore) Get(_ context.Context, idHash []byte) (Refresh, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.m[string(idHash)]
	if !ok {
		return Refresh{}, ErrNotFound
	}
	return cloneRefresh(r), nil
}

func (s *memoryRefreshStore) MarkUsed(_ context.Context, idHash []byte, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.m[string(idHash)]
	if !ok || r.UsedAt != nil {
		return false, nil
	}
	r.UsedAt = &at
	s.m[string(idHash)] = r
	return true, nil
}

func (s *memoryRefreshStore) DeleteFamily(_ context.Context, familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, r := range s.m {
		if r.FamilyID == familyID {
			delete(s.m, k)
		}
	}
	return nil
}
