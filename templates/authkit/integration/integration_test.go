//go:build integration

// End-to-end tests against the real stack (API + Postgres + migrations).
//
//	make integration      # starts the stack with docker compose, then runs this
//
// Cookies are handled by hand instead of net/http/cookiejar: the jar refuses
// to send Secure cookies over plain http://localhost, which would hide the
// very flags we want to assert. The tiny jar below mimics the browser rules
// that matter here (Max-Age < 0 deletes, Path scoping).
package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const password = "correct horse battery"

var (
	apiURL = getenv("API_URL", "http://localhost:8080")
	origin = getenv("WEB_ORIGIN", "http://localhost:3000") // what the SPA sends as Origin
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// ---- tiny browser-like client ---------------------------------------------

type client struct {
	t       *testing.T
	cookies map[string]*http.Cookie
	http    *http.Client
}

func newClient(t *testing.T) *client {
	return &client{
		t: t, cookies: map[string]*http.Cookie{},
		http: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

type reqOpt func(*http.Request)

func header(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }
func noHeader(k string) reqOpt  { return func(r *http.Request) { r.Header.Del(k) } }
func bearer(tok string) reqOpt  { return header("Authorization", "Bearer "+tok) }

// do sends a request the way the SPA does (Origin + CSRF header on POST).
func (c *client) do(method, path string, body any, opts ...reqOpt) (*http.Response, []byte) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, apiURL+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	for _, ck := range c.cookies {
		if strings.HasPrefix(path, ck.Path) { // cookie Path scoping
			req.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})
		}
	}
	for _, o := range opts {
		o(req)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v (is the stack up? try `make integration`)", method, path, err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck
		}
	}
	return res, out
}

func (c *client) mustStatus(want int, method, path string, body any, opts ...reqOpt) []byte {
	c.t.Helper()
	res, out := c.do(method, path, body, opts...)
	if res.StatusCode != want {
		c.t.Fatalf("%s %s = %d, want %d: %s", method, path, res.StatusCode, want, out)
	}
	return out
}

type creds map[string]string

func newUser(t *testing.T) (email string) {
	t.Helper()
	email = fmt.Sprintf("it-%s@example.com", uuid.NewString())
	newClient(t).mustStatus(201, "POST", "/register", creds{"email": email, "password": password})
	return email
}

func login(t *testing.T, c *client, mode, email string) (accessToken string) {
	t.Helper()
	out := c.mustStatus(200, "POST", "/"+mode+"/login", creds{"email": email, "password": password})
	if mode == "token" {
		var r struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(out, &r); err != nil || r.AccessToken == "" {
			t.Fatalf("no access token in %s", out)
		}
		return r.AccessToken
	}
	return ""
}

func accessFrom(t *testing.T, body []byte) string {
	t.Helper()
	var r struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		t.Fatalf("no access token in %s", body)
	}
	return r.AccessToken
}

func assertCookie(t *testing.T, c *client, name, path string, sameSite http.SameSite) *http.Cookie {
	t.Helper()
	ck := c.cookies[name]
	if ck == nil {
		t.Fatalf("cookie %q not set", name)
	}
	if !ck.HttpOnly || !ck.Secure || ck.SameSite != sameSite || ck.Path != path || ck.MaxAge <= 0 {
		t.Fatalf("cookie %q flags wrong: %+v", name, ck)
	}
	return ck
}

// ---- the flow from the plan -----------------------------------------------

// register -> login -> /me -> refresh -> reuse of the old refresh token is
// rejected -> logout
func TestTokenFlow(t *testing.T) {
	email := newUser(t) // register

	c := newClient(t)
	access1 := login(t, c, "token", email)
	refresh1 := assertCookie(t, c, "refresh_token", "/token", http.SameSiteStrictMode).Value

	// /me
	me := c.mustStatus(200, "GET", "/token/me", nil, bearer(access1))
	if !strings.Contains(string(me), `"roles":["user"]`) {
		t.Fatalf("/token/me = %s", me)
	}

	// refresh: rotates the cookie and returns a new access token
	out := c.mustStatus(200, "POST", "/token/refresh", nil)
	access2 := accessFrom(t, out)
	refresh2 := assertCookie(t, c, "refresh_token", "/token", http.SameSiteStrictMode).Value
	// Only the refresh token must change. The access JWT has no jti, so two
	// issued within the same second for the same user are byte-identical.
	if refresh2 == refresh1 {
		t.Fatal("refresh token did not rotate")
	}
	if strings.Contains(string(out), refresh2) {
		t.Fatal("refresh token leaked into the JSON body")
	}
	c.mustStatus(200, "GET", "/token/me", nil, bearer(access2))

	// reuse: someone replays the already-used first token
	thief := newClient(t)
	thief.cookies["refresh_token"] = &http.Cookie{Name: "refresh_token", Value: refresh1, Path: "/token"}
	if res, body := thief.do("POST", "/token/refresh", nil); res.StatusCode != 401 {
		t.Fatalf("replayed refresh token = %d, want 401: %s", res.StatusCode, body)
	}
	if _, still := thief.cookies["refresh_token"]; still {
		t.Fatal("server did not clear the cookie after reuse")
	}
	// the whole family is revoked: the victim's newest token is dead too
	if res, _ := c.do("POST", "/token/refresh", nil); res.StatusCode != 401 {
		t.Fatalf("newest refresh token after reuse = %d, want 401", res.StatusCode)
	}
	// access tokens are stateless, so this one stays valid until it expires
	// (at most 15 minutes): the price of not hitting the DB on every request
	c.mustStatus(200, "GET", "/token/me", nil, bearer(access2))

	// logout (idempotent, even though the family is already gone)
	c.mustStatus(204, "POST", "/token/logout", nil)
}

func TestTokenLogoutRevokesRefreshToken(t *testing.T) {
	email := newUser(t)
	c := newClient(t)
	login(t, c, "token", email)
	saved := c.cookies["refresh_token"].Value

	c.mustStatus(204, "POST", "/token/logout", nil)
	if _, ok := c.cookies["refresh_token"]; ok {
		t.Fatal("logout did not clear the cookie")
	}
	// even a copy of the cookie taken before logout no longer works
	copyOf := newClient(t)
	copyOf.cookies["refresh_token"] = &http.Cookie{Name: "refresh_token", Value: saved, Path: "/token"}
	if res, _ := copyOf.do("POST", "/token/refresh", nil); res.StatusCode != 401 {
		t.Fatalf("refresh after logout = %d, want 401", res.StatusCode)
	}
}

func TestSessionFlow(t *testing.T) {
	email := newUser(t)
	c := newClient(t)

	c.mustStatus(401, "GET", "/session/me", nil)
	login(t, c, "session", email)
	first := assertCookie(t, c, "sid", "/", http.SameSiteLaxMode).Value
	me := c.mustStatus(200, "GET", "/session/me", nil)
	if !strings.Contains(string(me), `"roles":["user"]`) {
		t.Fatalf("/session/me = %s", me)
	}

	// logging in again rotates the id and kills the old one (session fixation)
	login(t, c, "session", email)
	second := c.cookies["sid"].Value
	if second == first {
		t.Fatal("session id not rotated on login")
	}
	old := newClient(t)
	old.cookies["sid"] = &http.Cookie{Name: "sid", Value: first, Path: "/"}
	old.mustStatus(401, "GET", "/session/me", nil)

	c.mustStatus(204, "POST", "/session/logout", nil)
	if _, ok := c.cookies["sid"]; ok {
		t.Fatal("logout did not clear the cookie")
	}
	c.mustStatus(401, "GET", "/session/me", nil)
	// a copy of the cookie from before logout is dead server-side
	stolen := newClient(t)
	stolen.cookies["sid"] = &http.Cookie{Name: "sid", Value: second, Path: "/"}
	stolen.mustStatus(401, "GET", "/session/me", nil)
}

func TestSessionFixationAttemptFails(t *testing.T) {
	email := newUser(t)
	const planted = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	c := newClient(t)
	c.cookies["sid"] = &http.Cookie{Name: "sid", Value: planted, Path: "/"}

	login(t, c, "session", email)
	if c.cookies["sid"].Value == planted {
		t.Fatal("server adopted the client-supplied session id")
	}
	attacker := newClient(t)
	attacker.cookies["sid"] = &http.Cookie{Name: "sid", Value: planted, Path: "/"}
	attacker.mustStatus(401, "GET", "/session/me", nil)
}

func TestModesDoNotCrossAuthenticate(t *testing.T) {
	email := newUser(t)
	c := newClient(t)
	access := login(t, c, "token", email)
	login(t, c, "session", email)

	c.mustStatus(401, "GET", "/session/me", nil, bearer(access), header("Cookie", "")) // bearer is not a session
	c.mustStatus(401, "GET", "/token/me", nil)                                         // cookies are not a bearer token
}

// ---- hardening --------------------------------------------------------------

func TestCSRFAndCORS(t *testing.T) {
	email := newUser(t)
	body := creds{"email": email, "password": password}

	for _, path := range []string{"/session/login", "/token/login", "/token/refresh", "/session/logout", "/token/logout", "/register"} {
		c := newClient(t)
		res, _ := c.do("POST", path, body, noHeader("X-Requested-With"))
		if res.StatusCode != 403 || len(res.Cookies()) != 0 {
			t.Errorf("POST %s without CSRF header = %d (cookies %d), want 403 and none", path, res.StatusCode, len(res.Cookies()))
		}
		res, _ = c.do("POST", path, body, header("Origin", "http://evil.example"))
		if res.StatusCode != 403 || len(res.Cookies()) != 0 {
			t.Errorf("POST %s from evil origin = %d, want 403", path, res.StatusCode)
		}
	}

	// preflight from the real front-end origin is approved ...
	c := newClient(t)
	res, _ := c.do("OPTIONS", "/token/refresh", nil,
		header("Access-Control-Request-Method", "POST"),
		header("Access-Control-Request-Headers", "x-requested-with,content-type"))
	h := res.Header
	if res.StatusCode != 204 || h.Get("Access-Control-Allow-Origin") != origin ||
		h.Get("Access-Control-Allow-Credentials") != "true" ||
		!strings.Contains(strings.ToLower(h.Get("Access-Control-Allow-Headers")), "x-requested-with") {
		t.Fatalf("preflight = %d %v", res.StatusCode, h)
	}
	// ... from any other origin it is not
	res, _ = c.do("OPTIONS", "/token/refresh", nil,
		header("Origin", "http://evil.example"), header("Access-Control-Request-Method", "POST"))
	if res.StatusCode != 403 || res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("evil preflight = %d %v", res.StatusCode, res.Header)
	}
	// a real response lets the allowed origin read it, and never uses a wildcard
	res, _ = c.do("POST", "/session/login", body)
	if res.Header.Get("Access-Control-Allow-Origin") != origin || res.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("login response CORS headers: %v", res.Header)
	}
}

func TestLoginDoesNotRevealWhichAccountsExist(t *testing.T) {
	email := newUser(t)
	for _, mode := range []string{"session", "token"} {
		c := newClient(t)
		unknown, ub := c.do("POST", "/"+mode+"/login", creds{"email": "ghost-" + uuid.NewString() + "@example.com", "password": password})
		wrong, wb := c.do("POST", "/"+mode+"/login", creds{"email": email, "password": "not the password"})
		if unknown.StatusCode != 401 || wrong.StatusCode != 401 || !bytes.Equal(ub, wb) {
			t.Errorf("%s: unknown=%d %s / wrong=%d %s", mode, unknown.StatusCode, ub, wrong.StatusCode, wb)
		}
		if len(unknown.Cookies())+len(wrong.Cookies()) != 0 {
			t.Errorf("%s: failed login set a cookie", mode)
		}
	}
}

// A missing account must still pay for one password hash. This is a coarse
// guard against someone "optimising" the dummy hash away, not a precise timing
// measurement (that would be flaky): the median unknown-account login must take
// at least half as long as the median wrong-password login.
func TestUnknownAccountStillPaysForAHash(t *testing.T) {
	email := newUser(t)
	c := newClient(t)
	median := func(body creds) time.Duration {
		var ds []time.Duration
		for i := 0; i < 7; i++ {
			start := time.Now()
			c.do("POST", "/token/login", body)
			ds = append(ds, time.Since(start))
		}
		for i := range ds { // insertion sort, 7 items
			for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
				ds[j], ds[j-1] = ds[j-1], ds[j]
			}
		}
		return ds[len(ds)/2]
	}
	wrong := median(creds{"email": email, "password": "not the password"})
	unknown := median(creds{"email": "ghost-" + uuid.NewString() + "@example.com", "password": password})
	t.Logf("median wrong-password=%v unknown-account=%v", wrong, unknown)
	if unknown < wrong/2 {
		t.Fatalf("unknown account answered in %v vs %v: the dummy hash is not being verified", unknown, wrong)
	}
}

func TestAccessTokenForgery(t *testing.T) {
	email := newUser(t)
	c := newClient(t)
	real := login(t, c, "token", email)
	parts := strings.Split(real, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", real)
	}
	b64 := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	exp := time.Now().Add(time.Hour).Unix()
	adminClaims := fmt.Sprintf(`{"sub":"%s","roles":["admin"],"iss":"authkit","iat":%d,"exp":%d}`, uuid.NewString(), time.Now().Unix()-5, exp)

	for name, tok := range map[string]string{
		"alg=none":            b64(`{"alg":"none","typ":"JWT"}`) + "." + b64(adminClaims) + ".",
		"alg=none, fake sig":  b64(`{"alg":"none","typ":"JWT"}`) + "." + b64(adminClaims) + "." + parts[2],
		"payload swapped":     parts[0] + "." + b64(adminClaims) + "." + parts[2],
		"signature stripped":  parts[0] + "." + parts[1] + ".",
		"signature corrupted": parts[0] + "." + parts[1] + "." + strings.Repeat("A", len(parts[2])),
		"garbage":             "not-a-jwt",
	} {
		if res, body := c.do("GET", "/token/me", nil, bearer(tok)); res.StatusCode != 401 {
			t.Errorf("%s accepted: %d %s", name, res.StatusCode, body)
		}
	}
	c.mustStatus(200, "GET", "/token/me", nil, bearer(real)) // the real one still works
}

func TestRegisterValidation(t *testing.T) {
	email := newUser(t)
	c := newClient(t)
	for name, tc := range map[string]struct {
		body any
		want int
	}{
		"duplicate":       {creds{"email": email, "password": password}, 409},
		"duplicate, case": {creds{"email": strings.ToUpper(email), "password": password}, 409},
		"invalid email":   {creds{"email": "nope", "password": password}, 400},
		"short password":  {creds{"email": "a-" + email, "password": "short"}, 400},
		"unknown field":   {`{"email":"b-` + email + `","password":"` + password + `","role":"admin"}`, 400},
		"oversized body":  {`{"email":"c-` + email + `","password":"` + strings.Repeat("x", 70<<10) + `"}`, 413},
	} {
		var body any = tc.body
		if s, ok := body.(string); ok {
			body = json.RawMessage(s)
		}
		if res, out := c.do("POST", "/register", body); res.StatusCode != tc.want {
			t.Errorf("%s = %d, want %d: %s", name, res.StatusCode, tc.want, out)
		}
	}
}
