package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"authkit/password"
	"authkit/session"
	"authkit/store"
	"authkit/token"
)

// ---- fakes ---------------------------------------------------------------

type fakeUsers struct {
	mu    sync.Mutex
	byID  map[string]store.User
	index map[string]string // lower(email) -> id
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]store.User{}, index: map[string]string{}}
}

func (f *fakeUsers) Create(_ context.Context, email, hash string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return store.User{}, store.ErrInvalidEmail
	}
	k := strings.ToLower(email)
	if _, ok := f.index[k]; ok {
		return store.User{}, store.ErrEmailTaken
	}
	u := store.User{ID: uuid.NewString(), Email: email, PasswordHash: hash, Role: "user", CreatedAt: time.Now()}
	f.byID[u.ID], f.index[k] = u, u.ID
	return u, nil
}

func (f *fakeUsers) ByEmail(_ context.Context, email string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.index[strings.ToLower(email)]; ok {
		return f.byID[id], nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeUsers) ByID(_ context.Context, id string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeUsers) remove(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.byID[id]
	delete(f.byID, id)
	delete(f.index, strings.ToLower(u.Email))
}

func (f *fakeUsers) setRole(id, role string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.byID[id]
	u.Role = role
	f.byID[id] = u
}

// fakeHasher is fast and counts Verify calls, to prove login does the same
// amount of hashing work whether or not the account exists.
type fakeHasher struct{ verifies, hashes atomic.Int32 }

func (h *fakeHasher) Hash(pw string) (string, error) { h.hashes.Add(1); return "fake$" + pw, nil }
func (h *fakeHasher) Verify(pw, enc string) (bool, error) {
	h.verifies.Add(1)
	if !strings.HasPrefix(enc, "fake$") {
		return false, password.ErrInvalidHash
	}
	return enc == "fake$"+pw, nil
}

// ---- harness -------------------------------------------------------------

type env struct {
	h      http.Handler
	users  *fakeUsers
	hasher *fakeHasher
}

const testSecret = "0123456789abcdef0123456789abcdef"

func newEnv(t *testing.T) *env {
	t.Helper()
	users, hasher := newFakeUsers(), &fakeHasher{}
	access, err := token.NewAccess(token.AccessOptions{Secret: []byte(testSecret), Issuer: "authkit"})
	if err != nil {
		t.Fatal(err)
	}
	h, err := newHandler(deps{
		Users:  users,
		Hasher: hasher,
		Sessions: session.NewManager(session.NewMemoryStore(), session.Options{
			Roles: rolesFromStore(users),
		}),
		Access:           access,
		Refresh:          token.NewRefreshManager(token.NewMemoryRefreshStore(), token.RefreshOptions{TTL: refreshTTL}),
		RefreshCookieTTL: refreshTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &env{h: h, users: users, hasher: hasher}
}

func (e *env) call(method, path string, body any, mods ...func(*http.Request)) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, _ := json.Marshal(b)
		rd = bytes.NewReader(raw)
	}
	r := httptest.NewRequest(method, path, rd)
	for _, m := range mods {
		m(r)
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, r)
	return w
}

func withCookie(c *http.Cookie) func(*http.Request) {
	return func(r *http.Request) {
		if c != nil {
			r.AddCookie(c)
		}
	}
}
func withBearer(tok string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) }
}

func cookieNamed(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

type creds map[string]string

func (e *env) register(t *testing.T, email, pw string) {
	t.Helper()
	if w := e.call("POST", "/register", creds{"email": email, "password": pw}); w.Code != 201 {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
}

const pw = "correct horse battery"

// bigJSON is valid JSON larger than the body limit.
func bigJSON() string {
	return `{"email":"a@example.com","password":"` + strings.Repeat("x", 70<<10) + `"}`
}

// ---- /register -----------------------------------------------------------

func TestRegister(t *testing.T) {
	e := newEnv(t)
	w := e.call("POST", "/register", creds{"email": "a@example.com", "password": pw})
	if w.Code != 201 {
		t.Fatalf("code %d: %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), pw) || strings.Contains(strings.ToLower(w.Body.String()), "hash") {
		t.Fatalf("response leaks secrets: %s", w.Body)
	}
	u, _ := e.users.ByEmail(context.Background(), "a@example.com")
	if u.PasswordHash == "" || u.PasswordHash == pw {
		t.Fatalf("password stored as %q", u.PasswordHash)
	}
}

func TestRegister_Errors(t *testing.T) {
	e := newEnv(t)
	e.register(t, "dup@example.com", pw)
	for name, c := range map[string]struct {
		body any
		want int
	}{
		"duplicate":         {creds{"email": "dup@example.com", "password": pw}, 409},
		"duplicate, case":   {creds{"email": "DUP@example.com", "password": pw}, 409},
		"invalid email":     {creds{"email": "nope", "password": pw}, 400},
		"missing email":     {creds{"password": pw}, 400},
		"short password":    {creds{"email": "b@example.com", "password": "short"}, 400},
		"password too long": {creds{"email": "b@example.com", "password": strings.Repeat("x", 129)}, 400},
		"malformed json":    {"{not json", 400},
		"unknown field":     {`{"email":"b@example.com","password":"` + pw + `","role":"admin"}`, 400},
		"oversized body":    {bigJSON(), 413},
	} {
		if w := e.call("POST", "/register", c.body); w.Code != c.want {
			t.Errorf("%s: code %d, want %d (%s)", name, w.Code, c.want, strings.TrimSpace(w.Body.String()))
		}
	}
}

func TestRegister_CannotChooseRole(t *testing.T) {
	e := newEnv(t)
	e.call("POST", "/register", `{"email":"x@example.com","password":"`+pw+`","role":"admin"}`)
	e.register(t, "y@example.com", pw)
	u, _ := e.users.ByEmail(context.Background(), "y@example.com")
	if u.Role != "user" {
		t.Fatalf("role = %q", u.Role)
	}
}

// ---- login hardening -----------------------------------------------------

func TestLogin_FailureIsUniformAndDoesEqualWork(t *testing.T) {
	for _, path := range []string{"/session/login", "/token/login"} {
		e := newEnv(t)
		e.register(t, "real@example.com", pw)

		e.hasher.verifies.Store(0)
		unknown := e.call("POST", path, creds{"email": "ghost@example.com", "password": pw})
		if got := e.hasher.verifies.Load(); got != 1 {
			t.Errorf("%s unknown account: %d Verify calls, want 1 (dummy hash)", path, got)
		}
		e.hasher.verifies.Store(0)
		wrong := e.call("POST", path, creds{"email": "real@example.com", "password": "wrong password"})
		if got := e.hasher.verifies.Load(); got != 1 {
			t.Errorf("%s wrong password: %d Verify calls, want 1", path, got)
		}

		if unknown.Code != 401 || wrong.Code != 401 {
			t.Errorf("%s codes: unknown=%d wrong=%d, want 401", path, unknown.Code, wrong.Code)
		}
		if unknown.Body.String() != wrong.Body.String() {
			t.Errorf("%s bodies differ:\n unknown: %s\n wrong:   %s", path, unknown.Body, wrong.Body)
		}
		if len(unknown.Result().Cookies()) != 0 || len(wrong.Result().Cookies()) != 0 {
			t.Errorf("%s failed login set a cookie", path)
		}
	}
}

func TestLogin_DummyHashIsGeneratedOnce(t *testing.T) {
	e := newEnv(t)
	if got := e.hasher.hashes.Load(); got != 1 {
		t.Fatalf("Hash calls at startup = %d, want 1", got)
	}
}

func TestLogin_BadInput(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/session/login", "/token/login"} {
		for name, body := range map[string]any{
			"empty":     creds{},
			"no pw":     creds{"email": "a@example.com"},
			"bad json":  "{",
			"oversized": bigJSON(),
		} {
			if w := e.call("POST", path, body); w.Code != 400 && w.Code != 413 {
				t.Errorf("%s %s: code %d, want 400/413", path, name, w.Code)
			}
		}
	}
}

func TestLogin_EmailCaseInsensitive(t *testing.T) {
	e := newEnv(t)
	e.register(t, "Mixed@Example.com", pw)
	if w := e.call("POST", "/session/login", creds{"email": "mixed@example.com", "password": pw}); w.Code != 200 {
		t.Fatalf("code %d", w.Code)
	}
}

// ---- session mode --------------------------------------------------------

func TestSessionFlow(t *testing.T) {
	e := newEnv(t)
	e.register(t, "s@example.com", pw)

	if w := e.call("GET", "/session/me", nil); w.Code != 401 {
		t.Fatalf("me without cookie: %d", w.Code)
	}

	w := e.call("POST", "/session/login", creds{"email": "s@example.com", "password": pw})
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	c := cookieNamed(w, "sid")
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("bad session cookie: %+v", c)
	}

	w = e.call("GET", "/session/me", nil, withCookie(c))
	if w.Code != 200 {
		t.Fatalf("me: %d", w.Code)
	}
	var me struct {
		UserID string   `json:"user_id"`
		Roles  []string `json:"roles"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &me)
	if me.UserID == "" || len(me.Roles) != 1 || me.Roles[0] != "user" {
		t.Fatalf("me = %+v", me)
	}

	if w := e.call("POST", "/session/logout", nil, withCookie(c)); w.Code != 204 {
		t.Fatalf("logout: %d", w.Code)
	}
	if w := e.call("GET", "/session/me", nil, withCookie(c)); w.Code != 401 {
		t.Fatalf("me after logout: %d", w.Code)
	}
}

func TestSession_LoginRotatesID(t *testing.T) {
	e := newEnv(t)
	e.register(t, "s@example.com", pw)
	login := func(c *http.Cookie) *http.Cookie {
		w := e.call("POST", "/session/login", creds{"email": "s@example.com", "password": pw}, withCookie(c))
		return cookieNamed(w, "sid")
	}
	first := login(nil)
	second := login(first)
	if first.Value == second.Value {
		t.Fatal("session id not rotated")
	}
	if w := e.call("GET", "/session/me", nil, withCookie(first)); w.Code != 401 {
		t.Fatalf("old session still valid: %d", w.Code)
	}
}

func TestSession_RoleChangeVisibleImmediately(t *testing.T) {
	e := newEnv(t)
	e.register(t, "s@example.com", pw)
	w := e.call("POST", "/session/login", creds{"email": "s@example.com", "password": pw})
	c := cookieNamed(w, "sid")
	u, _ := e.users.ByEmail(context.Background(), "s@example.com")
	e.users.setRole(u.ID, "admin")

	w = e.call("GET", "/session/me", nil, withCookie(c))
	if !strings.Contains(w.Body.String(), "admin") {
		t.Fatalf("role change not reflected: %s", w.Body)
	}
}

func TestSession_DeletedUserIs401(t *testing.T) {
	e := newEnv(t)
	e.register(t, "s@example.com", pw)
	w := e.call("POST", "/session/login", creds{"email": "s@example.com", "password": pw})
	c := cookieNamed(w, "sid")
	u, _ := e.users.ByEmail(context.Background(), "s@example.com")
	e.users.remove(u.ID)

	if w := e.call("GET", "/session/me", nil, withCookie(c)); w.Code != 401 {
		t.Fatalf("code %d, want 401", w.Code)
	}
}

// ---- token mode ----------------------------------------------------------

type tokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (e *env) tokenLogin(t *testing.T, email string) (tokenResp, *http.Cookie) {
	t.Helper()
	w := e.call("POST", "/token/login", creds{"email": email, "password": pw})
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	var tr tokenResp
	if err := json.Unmarshal(w.Body.Bytes(), &tr); err != nil {
		t.Fatal(err)
	}
	return tr, cookieNamed(w, "refresh_token")
}

func TestTokenFlow(t *testing.T) {
	e := newEnv(t)
	e.register(t, "t@example.com", pw)

	tr, rc := e.tokenLogin(t, "t@example.com")
	if tr.TokenType != "Bearer" || tr.ExpiresIn < 800 || tr.ExpiresIn > 900 || tr.AccessToken == "" {
		t.Fatalf("token response: %+v", tr)
	}
	if rc == nil || !rc.HttpOnly || !rc.Secure || rc.SameSite != http.SameSiteStrictMode || rc.Path != "/token" {
		t.Fatalf("bad refresh cookie: %+v", rc)
	}

	w := e.call("GET", "/token/me", nil, withBearer(tr.AccessToken))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user"`) {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if w := e.call("GET", "/token/me", nil); w.Code != 401 {
		t.Fatalf("me without token: %d", w.Code)
	}
}

func TestToken_RefreshTokenOnlyInCookie(t *testing.T) {
	e := newEnv(t)
	e.register(t, "t@example.com", pw)
	w := e.call("POST", "/token/login", creds{"email": "t@example.com", "password": pw})
	rc := cookieNamed(w, "refresh_token")
	if strings.Contains(w.Body.String(), rc.Value) {
		t.Fatal("refresh token leaked into JSON body")
	}
}

func TestToken_RefreshRotatesAndDetectsReuse(t *testing.T) {
	e := newEnv(t)
	e.register(t, "t@example.com", pw)
	_, old := e.tokenLogin(t, "t@example.com")

	w := e.call("POST", "/token/refresh", nil, withCookie(old))
	if w.Code != 200 {
		t.Fatalf("refresh: %d %s", w.Code, w.Body)
	}
	fresh := cookieNamed(w, "refresh_token")
	var tr tokenResp
	_ = json.Unmarshal(w.Body.Bytes(), &tr)
	if fresh == nil || fresh.Value == old.Value || tr.AccessToken == "" {
		t.Fatalf("not rotated: cookie=%+v body=%s", fresh, w.Body)
	}
	if e.call("GET", "/token/me", nil, withBearer(tr.AccessToken)).Code != 200 {
		t.Fatal("new access token rejected")
	}

	// replay the old one: rejected, cookie cleared ...
	w = e.call("POST", "/token/refresh", nil, withCookie(old))
	if w.Code != 401 {
		t.Fatalf("replay: %d, want 401", w.Code)
	}
	if c := cookieNamed(w, "refresh_token"); c == nil || c.MaxAge >= 0 {
		t.Fatalf("cookie not cleared on failure: %+v", c)
	}
	// ... and the whole family is dead, including the newest token
	if w := e.call("POST", "/token/refresh", nil, withCookie(fresh)); w.Code != 401 {
		t.Fatalf("newest token after reuse: %d, want 401", w.Code)
	}
}

func TestToken_RefreshRejects(t *testing.T) {
	e := newEnv(t)
	if w := e.call("POST", "/token/refresh", nil); w.Code != 401 {
		t.Fatalf("no cookie: %d", w.Code)
	}
	bogus := &http.Cookie{Name: "refresh_token", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
	if w := e.call("POST", "/token/refresh", nil, withCookie(bogus)); w.Code != 401 {
		t.Fatalf("bogus cookie: %d", w.Code)
	}
}

func TestToken_RefreshForDeletedUserIs401(t *testing.T) {
	e := newEnv(t)
	e.register(t, "t@example.com", pw)
	_, rc := e.tokenLogin(t, "t@example.com")
	u, _ := e.users.ByEmail(context.Background(), "t@example.com")
	e.users.remove(u.ID)
	if w := e.call("POST", "/token/refresh", nil, withCookie(rc)); w.Code != 401 {
		t.Fatalf("code %d, want 401", w.Code)
	}
}

func TestToken_Logout(t *testing.T) {
	e := newEnv(t)
	e.register(t, "t@example.com", pw)
	_, rc := e.tokenLogin(t, "t@example.com")

	w := e.call("POST", "/token/logout", nil, withCookie(rc))
	if w.Code != 204 {
		t.Fatalf("logout: %d", w.Code)
	}
	if c := cookieNamed(w, "refresh_token"); c == nil || c.MaxAge >= 0 {
		t.Fatalf("cookie not cleared: %+v", c)
	}
	if w := e.call("POST", "/token/refresh", nil, withCookie(rc)); w.Code != 401 {
		t.Fatalf("refresh after logout: %d", w.Code)
	}
	if w := e.call("POST", "/token/logout", nil); w.Code != 204 {
		t.Fatalf("logout without cookie should be idempotent: %d", w.Code)
	}
}

// ---- modes are isolated --------------------------------------------------

func TestModesDoNotCrossAuthenticate(t *testing.T) {
	e := newEnv(t)
	e.register(t, "m@example.com", pw)
	tr, _ := e.tokenLogin(t, "m@example.com")
	sc := cookieNamed(e.call("POST", "/session/login", creds{"email": "m@example.com", "password": pw}), "sid")

	if w := e.call("GET", "/session/me", nil, withBearer(tr.AccessToken)); w.Code != 401 {
		t.Errorf("bearer accepted on session route: %d", w.Code)
	}
	if w := e.call("GET", "/token/me", nil, withCookie(sc)); w.Code != 401 {
		t.Errorf("session cookie accepted on token route: %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	e := newEnv(t)
	if w := e.call("GET", "/session/login", nil); w.Code != 405 {
		t.Fatalf("code %d, want 405", w.Code)
	}
}
