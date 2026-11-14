package session

import (
	"context"
	"log"
	"net/http"
)

// VerifyFunc checks credentials and returns the user's ID. Production code
// would look the user up and compare an argon2id hash (see 密碼雜湊).
type VerifyFunc func(username, password string) (userID string, ok bool)

// Config holds the cookie settings that differ between environments.
type Config struct {
	CookieName string
	Secure     bool // true on HTTPS; false only for http://localhost demos
}

// Auth wires login, logout and the guard middleware to one Store.
type Auth struct {
	store  *Store
	verify VerifyFunc
	cfg    Config
}

// NewAuth builds an Auth.
func NewAuth(store *Store, verify VerifyFunc, cfg Config) *Auth {
	return &Auth{store: store, verify: verify, cfg: cfg}
}

type ctxKey struct{}

// UserID returns the logged-in user's ID set by Require.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}

// cookie builds the one Set-Cookie we ever send. maxAge < 0 deletes it.
func (a *Auth) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     a.cfg.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// Login checks the form credentials and starts a session.
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	userID, ok := a.verify(r.FormValue("username"), r.FormValue("password"))
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Never keep an ID the client brought along: always issue a fresh one.
	if old, err := r.Cookie(a.cfg.CookieName); err == nil {
		a.store.Delete(old.Value)
	}
	sess, err := a.store.Create(userID)
	if err != nil {
		log.Printf("login: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, a.cookie(sess.ID, 0))
	w.WriteHeader(http.StatusNoContent)
}

// Logout deletes the server-side session and tells the browser to drop the cookie.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cfg.CookieName); err == nil {
		a.store.Delete(c.Value)
	}
	http.SetCookie(w, a.cookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

// Require rejects requests without a live session and passes the user ID down.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(a.cfg.CookieName)
		if err != nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		sess, err := a.store.Get(c.Value)
		if err != nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, sess.UserID)))
	})
}
