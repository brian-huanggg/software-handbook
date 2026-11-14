// Package session keeps login state on the server and hands the browser
// an opaque ID to carry in a cookie.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNotFound covers unknown, deleted and expired IDs alike, so callers
// cannot tell an attacker which guesses were close.
var ErrNotFound = errors.New("session: not found or expired")

// Session is the server-side record behind one login.
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store is an in-memory session store. A real deployment swaps it for
// Redis or a database so that every server instance sees the same data.
type Store struct {
	mu       sync.Mutex
	sessions map[string]Session
	idleTTL  time.Duration
	now      func() time.Time // replaced in tests
}

// NewStore returns a store whose sessions expire after idleTTL without use.
func NewStore(idleTTL time.Duration) *Store {
	return &Store{
		sessions: make(map[string]Session),
		idleTTL:  idleTTL,
		now:      time.Now,
	}
}

// newID returns 256 bits from the OS CSPRNG, base64url-encoded (43 chars).
func newID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: generate id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Create starts a new session for userID under a fresh random ID.
func (s *Store) Create(userID string) (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	now := s.now()
	sess := Session{ID: id, UserID: userID, CreatedAt: now, ExpiresAt: now.Add(s.idleTTL)}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess, nil
}

// Get returns the session and pushes its expiry back by idleTTL.
func (s *Store) Get(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	now := s.now()
	if !now.Before(sess.ExpiresAt) {
		delete(s.sessions, id)
		return Session{}, ErrNotFound
	}
	sess.ExpiresAt = now.Add(s.idleTTL)
	s.sessions[id] = sess
	return sess, nil
}

// Delete removes a session. Deleting an unknown ID is not an error.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// Sweep drops expired sessions that nobody came back for.
func (s *Store) Sweep() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	n := 0
	for id, sess := range s.sessions {
		if !now.Before(sess.ExpiresAt) {
			delete(s.sessions, id)
			n++
		}
	}
	return n
}
