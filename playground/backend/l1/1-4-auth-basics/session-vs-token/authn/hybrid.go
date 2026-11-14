package authn

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Pair is what a hybrid login hands the client.
type Pair struct {
	Access  string
	Refresh string
}

// Hybrid checks a short-lived access token without touching the store, and
// keeps only the refresh tokens on the server. Logging out deletes the
// refresh token, so the login ends once the access token runs out.
type Hybrid struct {
	access *Token

	mu      sync.Mutex
	refresh map[string]record
	ttl     time.Duration // refresh token lifetime
	now     func() time.Time
}

// NewHybrid returns an issuer with accessTTL-long access tokens and
// refreshTTL-long refresh tokens.
func NewHybrid(key []byte, accessTTL, refreshTTL time.Duration) *Hybrid {
	return &Hybrid{
		access:  NewToken(key, accessTTL),
		refresh: make(map[string]record),
		ttl:     refreshTTL,
		now:     time.Now,
	}
}

func (h *Hybrid) Login(userID string) (Pair, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.issue(userID)
}

// issue must be called with h.mu held.
func (h *Hybrid) issue(userID string) (Pair, error) {
	a, err := h.access.Login(userID)
	if err != nil {
		return Pair{}, err
	}
	id := rand.Text() + rand.Text() // 2 x 128 bits from the OS CSPRNG
	h.refresh[id] = record{userID: userID, expires: h.now().Add(h.ttl)}
	return Pair{Access: a, Refresh: id}, nil
}

// Authenticate verifies the access token only; no store read.
func (h *Hybrid) Authenticate(access string) (string, error) {
	return h.access.Authenticate(access)
}

// Refresh trades a refresh token for a new pair and deletes the old one
// (rotation), so a refresh token works exactly once.
func (h *Hybrid) Refresh(refresh string) (Pair, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.refresh[refresh]
	delete(h.refresh, refresh)
	if !ok || !h.now().Before(r.expires) {
		return Pair{}, ErrInvalid
	}
	p, err := h.issue(r.userID)
	if err != nil {
		return Pair{}, fmt.Errorf("authn: refresh: %w", err)
	}
	return p, nil
}

// Logout revokes the refresh token. Access tokens already handed out stay
// valid until they expire.
func (h *Hybrid) Logout(refresh string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.refresh, refresh)
	return nil
}
