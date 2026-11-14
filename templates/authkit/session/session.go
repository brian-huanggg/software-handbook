// Package session implements cookie-based server-side sessions.
//
// The raw session ID exists only in the client's cookie. Stores see only its
// SHA-256 hash, so a leaked database cannot be replayed as valid sessions.
package session

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound  = errors.New("session: not found")
	ErrDuplicate = errors.New("session: id hash already exists")
)

type Session struct {
	IDHash    []byte // SHA-256 of the raw session ID
	UserID    string
	ExpiresAt time.Time
}

// Store persists sessions by ID hash. Implementations do not enforce expiry:
// Get returns expired rows and Manager decides.
type Store interface {
	// Create returns ErrDuplicate if the hash already exists.
	Create(ctx context.Context, s Session) error
	// Get returns ErrNotFound if there is no such session.
	Get(ctx context.Context, idHash []byte) (Session, error)
	// Delete is idempotent: deleting a missing session is not an error.
	Delete(ctx context.Context, idHash []byte) error
	// DeleteExpired removes sessions whose ExpiresAt is before now and returns
	// how many it removed.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
