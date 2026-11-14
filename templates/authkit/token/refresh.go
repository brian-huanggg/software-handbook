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
}

type RefreshStore interface {
	// Create returns ErrDuplicate if the hash already exists.
	Create(ctx context.Context, r Refresh) error
	// Get returns ErrNotFound if there is no such token.
	Get(ctx context.Context, idHash []byte) (Refresh, error)
	// MarkUsed atomically sets used_at if it is still unset and reports whether
	// this call did so. Rotation relies on exactly one concurrent caller winning.
	MarkUsed(ctx context.Context, idHash []byte, at time.Time) (bool, error)
	// DeleteFamily removes every token of the family. Missing family is not an error.
	DeleteFamily(ctx context.Context, familyID string) error
}

type RefreshOptions struct {
	TTL time.Duration // lifetime of each token; default 7 days
	Now func() time.Time
}

// RefreshManager issues, rotates and revokes opaque refresh tokens.
type RefreshManager struct {
	store RefreshStore
	ttl   time.Duration
	now   func() time.Time
}

func NewRefreshManager(s RefreshStore, o RefreshOptions) *RefreshManager {
	if o.TTL <= 0 {
		o.TTL = 7 * 24 * time.Hour
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &RefreshManager{store: s, ttl: o.TTL, now: o.Now}
}

func hashToken(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

func newRaw() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *RefreshManager) issue(ctx context.Context, userID, familyID string) (string, error) {
	raw, err := newRaw()
	if err != nil {
		return "", err
	}
	err = m.store.Create(ctx, Refresh{
		IDHash: hashToken(raw), UserID: userID, FamilyID: familyID,
		ExpiresAt: m.now().Add(m.ttl),
	})
	if err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return raw, nil
}

// Start begins a new family (a login) and returns its first raw token.
func (m *RefreshManager) Start(ctx context.Context, userID string) (string, error) {
	return m.issue(ctx, userID, uuid.NewString())
}

// Rotate consumes raw and returns its replacement plus the owning user ID.
//
// Presenting a token that was already rotated is treated as theft: the whole
// family is revoked and ErrReuseDetected is returned. A lost race between two
// concurrent uses of the same token is handled the same way, so a legitimate
// client that double-submits is logged out and must sign in again.
func (m *RefreshManager) Rotate(ctx context.Context, raw string) (newRaw, userID string, err error) {
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
	if !m.now().Before(cur.ExpiresAt) {
		return "", "", ErrInvalidRefresh
	}
	if cur.UsedAt != nil {
		return "", "", m.revokeForReuse(ctx, cur.FamilyID)
	}
	won, err := m.store.MarkUsed(ctx, h, m.now())
	if err != nil {
		return "", "", fmt.Errorf("mark refresh token used: %w", err)
	}
	if !won {
		return "", "", m.revokeForReuse(ctx, cur.FamilyID)
	}
	newRaw, err = m.issue(ctx, cur.UserID, cur.FamilyID)
	if err != nil {
		return "", "", err
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
