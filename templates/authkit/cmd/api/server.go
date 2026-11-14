package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
	"github.com/brian-huanggg/software-handbook/templates/authkit/password"
	"github.com/brian-huanggg/software-handbook/templates/authkit/session"
	"github.com/brian-huanggg/software-handbook/templates/authkit/store"
	"github.com/brian-huanggg/software-handbook/templates/authkit/token"
)

const (
	maxBodyBytes     = 64 << 10 // credentials are tiny; refuse anything big
	minPasswordBytes = 8
	maxPasswordBytes = 128 // bounds the work a single hash can be made to do

	refreshCookieName = "refresh_token"
	refreshCookiePath = "/token" // only sent to /token/*, never to the rest of the API
)

type deps struct {
	Users            store.UserStore
	Hasher           password.Hasher
	Sessions         *session.Manager
	Access           *token.Access
	Refresh          *token.RefreshManager
	RefreshCookieTTL time.Duration
	InsecureCookies  bool
	AllowedOrigins   []string // CORS + CSRF; empty means same-origin only

	// Brute-force limits. Nil disables a limit (tests). AuthIPLimit applies
	// per client IP to /register and both logins; LoginEmailLimit applies per
	// email to both logins, wherever the attempts come from.
	AuthIPLimit     *middleware.Limiter
	LoginEmailLimit *middleware.Limiter
}

type server struct {
	deps
	// dummyHash is verified against when the account does not exist, so an
	// unknown email costs the same as a wrong password.
	dummyHash string
}

// rolesFromStore resolves roles for session Principals. A deleted user means
// the session is no longer valid.
func rolesFromStore(users store.UserStore) func(context.Context, string) ([]string, error) {
	return func(ctx context.Context, userID string) ([]string, error) {
		u, err := users.ByID(ctx, userID)
		if errors.Is(err, store.ErrNotFound) {
			return nil, authn.ErrUnauthenticated
		}
		if err != nil {
			return nil, err
		}
		return []string{u.Role}, nil
	}
}

func newHandler(d deps) (http.Handler, error) {
	dummy, err := d.Hasher.Hash(uuid.NewString())
	if err != nil {
		return nil, fmt.Errorf("make dummy hash: %w", err)
	}
	s := &server{deps: d, dummyHash: dummy}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	// Every endpoint that runs a password hash is limited per client IP.
	perIP := middleware.RateLimit(d.AuthIPLimit, middleware.ClientIP)
	mux.Handle("POST /register", perIP(http.HandlerFunc(s.register)))

	// Session mode: server-side state, cookie credential.
	requireSession := middleware.Require(d.Sessions)
	mux.Handle("POST /session/login", perIP(http.HandlerFunc(s.sessionLogin)))
	mux.HandleFunc("POST /session/logout", s.sessionLogout)
	mux.Handle("GET /session/me", requireSession(http.HandlerFunc(s.me)))

	// Token mode: stateless access JWT in a header, rotating refresh cookie.
	requireBearer := middleware.Require(token.NewBearer(d.Access))
	mux.Handle("POST /token/login", perIP(http.HandlerFunc(s.tokenLogin)))
	mux.HandleFunc("POST /token/refresh", s.tokenRefresh)
	mux.HandleFunc("POST /token/logout", s.tokenLogout)
	mux.Handle("GET /token/me", requireBearer(http.HandlerFunc(s.me)))

	// CORS is outermost so preflights are answered before the CSRF check.
	return middleware.CORS(d.AllowedOrigins)(middleware.CSRF(d.AllowedOrigins)(mux)), nil
}

// ---- helpers -------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	slog.ErrorContext(r.Context(), msg, "err", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// readCredentials decodes a strict, size-limited JSON body. It writes the
// error response itself and reports whether decoding succeeded.
func readCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var c credentials
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
		}
		return credentials{}, false
	}
	return c, true
}

// ---- register ------------------------------------------------------------

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	c, ok := readCredentials(w, r)
	if !ok {
		return
	}
	if c.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if n := len(c.Password); n < minPasswordBytes || n > maxPasswordBytes {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("password must be %d-%d bytes", minPasswordBytes, maxPasswordBytes))
		return
	}
	hash, err := s.Hasher.Hash(c.Password)
	if err != nil {
		serverError(w, r, "hash password", err)
		return
	}
	_, err = s.Users.Create(r.Context(), c.Email, hash)
	switch {
	case errors.Is(err, store.ErrEmailTaken):
		// Answer exactly like a new account, so /register cannot be used to
		// find out who has one. Both paths hash once and hit the DB once.
		// A real app emails the owner here ("someone tried to sign up").
		slog.InfoContext(r.Context(), "register: email already taken")
		writeRegistered(w)
	case errors.Is(err, store.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid email")
	case err != nil:
		serverError(w, r, "create user", err)
	default:
		writeRegistered(w)
	}
}

// writeRegistered is the single response for both a new and an existing
// account. It deliberately carries no user ID.
func writeRegistered(w http.ResponseWriter) {
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "registration accepted"})
}

// ---- login (shared) ------------------------------------------------------

var errInvalidCredentials = errors.New("invalid credentials")

// checkCredentials always performs exactly one password verification. For an
// unknown email it verifies against dummyHash, so response time does not
// reveal whether the account exists.
func (s *server) checkCredentials(ctx context.Context, email, pw string) (store.User, error) {
	u, err := s.Users.ByEmail(ctx, email)
	found := err == nil
	hash := u.PasswordHash
	switch {
	case errors.Is(err, store.ErrNotFound):
		hash = s.dummyHash
	case err != nil:
		return store.User{}, err
	}
	match, err := s.Hasher.Verify(pw, hash)
	if err != nil {
		if found {
			return store.User{}, fmt.Errorf("verify stored hash: %w", err)
		}
		return store.User{}, errInvalidCredentials
	}
	if !found || !match {
		return store.User{}, errInvalidCredentials
	}
	return u, nil
}

// login decodes and checks credentials, writing the failure response itself.
func (s *server) login(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	c, ok := readCredentials(w, r)
	if !ok {
		return store.User{}, false
	}
	if c.Email == "" || c.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return store.User{}, false
	}
	// Per account, whether or not it exists, so the limit itself reveals
	// nothing. The trade-off: an attacker can slow down one victim's logins.
	if ok, wait := s.LoginEmailLimit.Allow(strings.ToLower(strings.TrimSpace(c.Email))); !ok {
		middleware.RetryAfter(w, wait)
		writeError(w, http.StatusTooManyRequests, "too many login attempts; try again later")
		return store.User{}, false
	}
	u, err := s.checkCredentials(r.Context(), c.Email, c.Password)
	switch {
	case errors.Is(err, errInvalidCredentials):
		// One message for unknown account and wrong password.
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return store.User{}, false
	case err != nil:
		serverError(w, r, "check credentials", err)
		return store.User{}, false
	}
	return u, true
}

// ---- session mode --------------------------------------------------------

func (s *server) sessionLogin(w http.ResponseWriter, r *http.Request) {
	u, ok := s.login(w, r)
	if !ok {
		return
	}
	if err := s.Sessions.Login(w, r, u.ID); err != nil {
		serverError(w, r, "start session", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": u.ID})
}

func (s *server) sessionLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.Sessions.Logout(w, r); err != nil {
		serverError(w, r, "end session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- token mode ----------------------------------------------------------

func (s *server) refreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     refreshCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !s.InsecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
}

func (s *server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, s.refreshCookie("", -1))
}

func (s *server) writeAccess(w http.ResponseWriter, u store.User) error {
	tok, exp, err := s.Access.Issue(authn.Principal{UserID: u.ID, Roles: []string{u.Role}})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(exp).Round(time.Second).Seconds()),
	})
	return nil
}

func (s *server) tokenLogin(w http.ResponseWriter, r *http.Request) {
	u, ok := s.login(w, r)
	if !ok {
		return
	}
	raw, err := s.Refresh.Start(r.Context(), u.ID)
	if err != nil {
		serverError(w, r, "start refresh family", err)
		return
	}
	// Cookie first: headers must be set before the body is written.
	http.SetCookie(w, s.refreshCookie(raw, int(s.RefreshCookieTTL.Seconds())))
	if err := s.writeAccess(w, u); err != nil {
		serverError(w, r, "issue access token", err)
	}
}

func (s *server) tokenRefresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	newRaw, userID, err := s.Refresh.Rotate(r.Context(), c.Value)
	switch {
	case errors.Is(err, token.ErrReuseDetected):
		slog.WarnContext(r.Context(), "refresh token reuse detected; family revoked")
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	case errors.Is(err, token.ErrInvalidRefresh):
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	case err != nil:
		serverError(w, r, "rotate refresh token", err)
		return
	}

	u, err := s.Users.ByID(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) { // account deleted since login
		_ = s.Refresh.Revoke(r.Context(), newRaw)
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err != nil {
		serverError(w, r, "load user", err)
		return
	}
	http.SetCookie(w, s.refreshCookie(newRaw, int(s.RefreshCookieTTL.Seconds())))
	if err := s.writeAccess(w, u); err != nil {
		serverError(w, r, "issue access token", err)
	}
}

func (s *server) tokenLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil {
		if err := s.Refresh.Revoke(r.Context(), c.Value); err != nil {
			serverError(w, r, "revoke refresh family", err)
			return
		}
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---- protected -----------------------------------------------------------

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	p, _ := authn.FromContext(r.Context())
	roles := p.Roles
	if roles == nil {
		roles = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": p.UserID, "roles": roles})
}
