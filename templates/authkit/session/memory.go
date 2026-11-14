package session

import (
	"context"
	"sync"
	"time"
)

type memoryStore struct {
	mu sync.RWMutex
	m  map[string]Session // key: string(IDHash)
}

// NewMemoryStore returns an in-process Store for tests and single-node dev.
func NewMemoryStore() Store { return &memoryStore{m: make(map[string]Session)} }

func clone(s Session) Session {
	s.IDHash = append([]byte(nil), s.IDHash...)
	return s
}

func (s *memoryStore) Create(_ context.Context, sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := string(sess.IDHash)
	if _, ok := s.m[k]; ok {
		return ErrDuplicate
	}
	s.m[k] = clone(sess)
	return nil
}

func (s *memoryStore) Get(_ context.Context, idHash []byte) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.m[string(idHash)]
	if !ok {
		return Session{}, ErrNotFound
	}
	return clone(sess), nil
}

func (s *memoryStore) Delete(_ context.Context, idHash []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, string(idHash))
	return nil
}

func (s *memoryStore) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, sess := range s.m {
		if sess.ExpiresAt.Before(now) {
			delete(s.m, k)
			n++
		}
	}
	return n, nil
}
