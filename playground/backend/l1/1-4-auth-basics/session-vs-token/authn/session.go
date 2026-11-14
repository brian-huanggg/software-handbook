package authn

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Session keeps one record per login on the server. The credential is only
// a random lookup key, so deleting the record ends the login immediately.
type Session struct {
	mu      sync.Mutex
	records map[string]record
	ttl     time.Duration
	now     func() time.Time // replaced in tests
	lookups int
}

type record struct {
	userID  string
	expires time.Time
}

// NewSession returns a store whose sessions expire ttl after login.
func NewSession(ttl time.Duration) *Session {
	return &Session{records: make(map[string]record), ttl: ttl, now: time.Now}
}

func (s *Session) Login(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("authn: generate session id: %w", err)
	}
	id := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = record{userID: userID, expires: s.now().Add(s.ttl)}
	return id, nil
}

func (s *Session) Authenticate(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookups++ // every request pays one store read
	r, ok := s.records[id]
	if !ok || !s.now().Before(r.expires) {
		delete(s.records, id)
		return "", ErrInvalid
	}
	return r.userID, nil
}

func (s *Session) Logout(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
	return nil
}

// Lookups reports how many times Authenticate read the store.
func (s *Session) Lookups() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lookups
}
