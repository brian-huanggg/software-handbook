package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("token: not found")
	ErrDuplicate = errors.New("token: id hash already exists")

	// ErrInvalidRefresh covers unknown, malformed, expired and revoked tokens.
	ErrInvalidRefresh = errors.New("token: invalid refresh token")
	// ErrReuseDetected means an already-rotated token was presented again.
	// The whole family has been revoked. It satisfies errors.Is(_, ErrInvalidRefresh).
	ErrReuseDetected = fmt.Errorf("%w: reuse detected", ErrInvalidRefresh)
)

// Refresh is a stored refresh token. IDHash is SHA-256 of the raw token.
type Refresh struct {
	IDHash    []byte
	UserID    string
	FamilyID  string // all tokens descending from one login share it
	UsedAt    *time.Time
	ExpiresAt time.Time
	// FamilyExpiresAt is the absolute end of the login chain, fixed at Start.
	// Rotation never issues a token that expires after it.
	FamilyExpiresAt time.Time
}

type RefreshStore interface {
	// Create returns ErrDuplicate if the hash already exists.
	Create(ctx context.Context, r Refresh) error
	// Get returns ErrNotFound if there is no such token.
	Get(ctx context.Context, idHash []byte) (Refresh, error)
	// Rotate atomically marks oldHash used (only if it exists and is still
	// unused) and stores next, in one all-or-nothing step. It reports whether
	// this call did so; on false or on error nothing is changed. Rotation
	// relies on exactly one concurrent caller winning.
	Rotate(ctx context.Context, oldHash []byte, usedAt time.Time, next Refresh) (bool, error)
	// DeleteFamily removes every token of the family. Missing family is not an error.
	DeleteFamily(ctx context.Context, familyID string) error
	// DeleteExpired removes tokens whose ExpiresAt is before now (ExpiresAt
	// never passes FamilyExpiresAt) and returns how many it removed. Expired tokens are rejected before reuse
	// detection runs, so purging them loses nothing.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type RefreshOptions struct {
	TTL       time.Duration // idle lifetime of each token; default 7 days
	FamilyTTL time.Duration // absolute lifetime of a login chain; default 30 days
	Now       func() time.Time
}

// RefreshManager issues, rotates and revokes opaque refresh tokens.
type RefreshManager struct {
	store     RefreshStore
	ttl       time.Duration
	familyTTL time.Duration
	now       func() time.Time
}

func NewRefreshManager(s RefreshStore, o RefreshOptions) *RefreshManager {
	if o.TTL <= 0 {
		o.TTL = 7 * 24 * time.Hour
	}
	if o.FamilyTTL <= 0 {
		o.FamilyTTL = 30 * 24 * time.Hour
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &RefreshManager{store: s, ttl: o.TTL, familyTTL: o.FamilyTTL, now: o.Now}
}

func hashToken(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// next builds a fresh token record for the family. Its expiry never passes
// familyExpires.
func (m *RefreshManager) next(userID, familyID string, familyExpires time.Time) (raw string, r Refresh, err error) {
	raw, err = randomToken()
	if err != nil {
		return "", Refresh{}, err
	}
	exp := m.now().Add(m.ttl)
	if exp.After(familyExpires) {
		exp = familyExpires
	}
	return raw, Refresh{
		IDHash: hashToken(raw), UserID: userID, FamilyID: familyID,
		ExpiresAt: exp, FamilyExpiresAt: familyExpires,
	}, nil
}

// Start begins a new family (a login) and returns its first raw token.
func (m *RefreshManager) Start(ctx context.Context, userID string) (string, error) {
	raw, r, err := m.next(userID, uuid.NewString(), m.now().Add(m.familyTTL))
	if err != nil {
		return "", err
	}
	if err := m.store.Create(ctx, r); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return raw, nil
}

// Rotate consumes raw and returns its replacement plus the owning user ID.
//
// Presenting a token that was already rotated is treated as theft: the whole
// family is revoked and ErrReuseDetected is returned. A lost race between two
// concurrent uses of the same token is handled the same way, so a legitimate
// client that double-submits is logged out and must sign in again.
//
// Consuming the old token and storing its replacement is one atomic store
// operation: a failure leaves the old token unused, so the client can retry
// instead of tripping reuse detection.
func (m *RefreshManager) Rotate(ctx context.Context, raw string) (string, string, error) {
	if raw == "" {
		return "", "", ErrInvalidRefresh
	}
	h := hashToken(raw)
	cur, err := m.store.Get(ctx, h)
	if errors.Is(err, ErrNotFound) {
		return "", "", ErrInvalidRefresh
	}
	if err != nil {
		return "", "", fmt.Errorf("get refresh token: %w", err)
	}
	if now := m.now(); !now.Before(cur.ExpiresAt) || !now.Before(cur.FamilyExpiresAt) {
		return "", "", ErrInvalidRefresh
	}
	if cur.UsedAt != nil {
		return "", "", m.revokeForReuse(ctx, cur.FamilyID)
	}
	newRaw, next, err := m.next(cur.UserID, cur.FamilyID, cur.FamilyExpiresAt)
	if err != nil {
		return "", "", err
	}
	won, err := m.store.Rotate(ctx, h, m.now(), next)
	if err != nil {
		return "", "", fmt.Errorf("rotate refresh token: %w", err)
	}
	if !won {
		return "", "", m.revokeForReuse(ctx, cur.FamilyID)
	}
	return newRaw, cur.UserID, nil
}

func (m *RefreshManager) revokeForReuse(ctx context.Context, familyID string) error {
	if err := m.store.DeleteFamily(ctx, familyID); err != nil {
		return fmt.Errorf("revoke family after reuse: %w", err)
	}
	return ErrReuseDetected
}

// Revoke ends the family that raw belongs to (logout). Unknown tokens are a no-op.
func (m *RefreshManager) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	cur, err := m.store.Get(ctx, hashToken(raw))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}
	return m.store.DeleteFamily(ctx, cur.FamilyID)
}

// Purge deletes expired tokens. Run it periodically; see DeleteExpired.
func (m *RefreshManager) Purge(ctx context.Context) (int64, error) {
	return m.store.DeleteExpired(ctx, m.now())
}
