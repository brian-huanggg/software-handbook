package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
)

const idBytes = 32

type Options struct {
	TTL        time.Duration // session lifetime; default 24h
	CookieName string        // default "sid"
	// InsecureCookies drops the Secure flag so cookies work over plain HTTP.
	// Dev only. Browsers other than Safari already accept Secure cookies on
	// http://localhost, so you usually do not need it.
	InsecureCookies bool
	// Roles resolves a user's roles for the Principal. Nil means no roles.
	Roles func(ctx context.Context, userID string) ([]string, error)
	Now   func() time.Time // test hook; default time.Now
}

// Manager issues, validates and revokes session cookies. It implements
// authn.Authenticator.
type Manager struct {
	store Store
	opts  Options
}

var _ authn.Authenticator = (*Manager)(nil)

func NewManager(s Store, o Options) *Manager {
	if o.TTL <= 0 {
		o.TTL = 24 * time.Hour
	}
	if o.CookieName == "" {
		o.CookieName = "sid"
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Manager{store: s, opts: o}
}

func hashID(id string) []byte {
	h := sha256.Sum256([]byte(id))
	return h[:]
}

func newID() (string, error) {
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Login creates a session for userID and sets the cookie. It always issues a
// brand-new ID and revokes any session the request already carried, so an ID
// planted before login (session fixation) never becomes authenticated.
func (m *Manager) Login(w http.ResponseWriter, r *http.Request, userID string) error {
	if c, err := r.Cookie(m.opts.CookieName); err == nil && c.Value != "" {
		if err := m.store.Delete(r.Context(), hashID(c.Value)); err != nil {
			return fmt.Errorf("revoke previous session: %w", err)
		}
	}
	id, err := newID()
	if err != nil {
		return err
	}
	exp := m.opts.Now().Add(m.opts.TTL)
	if err := m.store.Create(r.Context(), Session{IDHash: hashID(id), UserID: userID, ExpiresAt: exp}); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	http.SetCookie(w, m.cookie(id, int(m.opts.TTL.Seconds()), exp))
	return nil
}

// Logout revokes the request's session (if any) and clears the cookie.
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, m.cookie("", -1, time.Unix(0, 0)))
	c, err := r.Cookie(m.opts.CookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	return m.store.Delete(r.Context(), hashID(c.Value))
}

// Authenticate resolves the session cookie to a Principal. Every kind of
// invalid cookie yields authn.ErrUnauthenticated; other errors are infra
// failures.
func (m *Manager) Authenticate(r *http.Request) (authn.Principal, error) {
	c, err := r.Cookie(m.opts.CookieName)
	if err != nil || c.Value == "" {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	sess, err := m.store.Get(r.Context(), hashID(c.Value))
	if errors.Is(err, ErrNotFound) {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	if err != nil {
		return authn.Principal{}, fmt.Errorf("get session: %w", err)
	}
	if !m.opts.Now().Before(sess.ExpiresAt) {
		_ = m.store.Delete(r.Context(), sess.IDHash)
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	p := authn.Principal{UserID: sess.UserID}
	if m.opts.Roles != nil {
		if p.Roles, err = m.opts.Roles(r.Context(), sess.UserID); err != nil {
			return authn.Principal{}, fmt.Errorf("load roles: %w", err)
		}
	}
	return p, nil
}

// Purge deletes expired sessions. Authenticate only removes an expired
// session when its cookie comes back, so run this periodically.
func (m *Manager) Purge(ctx context.Context) (int64, error) {
	return m.store.DeleteExpired(ctx, m.opts.Now())
}

func (m *Manager) cookie(value string, maxAge int, exp time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     m.opts.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  exp,
		HttpOnly: true,
		Secure:   !m.opts.InsecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}
