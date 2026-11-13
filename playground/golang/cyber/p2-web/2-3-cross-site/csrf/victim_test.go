package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"handbook/playground/golang/cyber/csrf/defense"
)

const evilURL = "http://evil.test:9666/phish"

func newTestVictim(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	h, err := newVictim(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// send builds a request as alice's browser would, with caller-chosen headers.
func send(h http.Handler, method, target, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// crossSitePost is the request alice's browser sends when evil.test auto-submits
// a form to victim.test and her session cookie rides along.
func crossSitePost(h http.Handler) *httptest.ResponseRecorder {
	return send(h, http.MethodPost, "/settings/target", "url="+url.QueryEscape(evilURL), map[string]string{
		"Cookie":         "session=" + sessionValue,
		"Origin":         "http://evil.test:9666",
		"Sec-Fetch-Site": "cross-site",
	})
}

func targetOf(h http.Handler) string {
	return send(h, http.MethodGet, "/a1", "", nil).Header().Get("Location")
}

func TestForgedPostSucceedsWithoutDefense(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "none"})
	if rec := crossSitePost(h); rec.Code != http.StatusSeeOther {
		t.Fatalf("forged POST status = %d, want 303", rec.Code)
	}
	if got := targetOf(h); got != evilURL {
		t.Fatalf("/a1 -> %q, want the attacker's %q", got, evilURL)
	}
}

func TestWithoutCookieNothingHappens(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "none"})
	rec := send(h, http.MethodPost, "/settings/target", "url="+url.QueryEscape(evilURL), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: the attack depends on the cookie", rec.Code)
	}
}

func TestTokenBlocksForgedPost(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "token"})
	if rec := crossSitePost(h); rec.Code != http.StatusForbidden {
		t.Fatalf("forged POST status = %d, want 403", rec.Code)
	}
	if got := targetOf(h); got != initialTarget {
		t.Fatalf("/a1 -> %q, want unchanged %q", got, initialTarget)
	}
}

func TestTokenAllowsOwnForm(t *testing.T) {
	// The page's own form embeds the token; the attacker cannot read that page
	// cross-origin, so only the real form can send it.
	page := send(newTestVictim(t, Config{Defense: "token"}), http.MethodGet, "/settings", "", map[string]string{
		"Cookie": "session=" + sessionValue,
	})
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `name="csrf_token"`) {
		t.Fatalf("settings page lacks the token field: %d", page.Code)
	}
}

func TestTokenFromOwnFormIsAccepted(t *testing.T) {
	h, key := newTokenVictim(t)
	body := url.Values{"url": {"https://sho.rt/new"}, defense.FormField: {defense.TokenFor(key, sessionValue)}}
	rec := send(h, http.MethodPost, "/settings/target", body.Encode(), map[string]string{
		"Cookie":         "session=" + sessionValue,
		"Origin":         "http://victim.test:8080",
		"Sec-Fetch-Site": "same-origin",
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("own form status = %d, want 303", rec.Code)
	}
	if got := targetOf(h); got != "https://sho.rt/new" {
		t.Fatalf("/a1 -> %q, want the new target", got)
	}
}

// newTokenVictim builds the token-defended victim and returns its key so a
// test can mint the token the real form would carry.
func newTokenVictim(t *testing.T) (http.Handler, []byte) {
	t.Helper()
	key := []byte("test-key-test-key-test-key-test!")
	v := &victim{target: initialTarget, key: key, logger: log.New(io.Discard, "", 0)}
	mux := http.NewServeMux()
	mux.Handle("POST /settings/target", requireSession(http.HandlerFunc(v.setTarget)))
	mux.HandleFunc("GET /a1", v.redirect)
	return defense.Token(key, sessionOf)(mux), key
}

func TestCrossSiteBlocksForgedPost(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "crosssite"})
	if rec := crossSitePost(h); rec.Code != http.StatusForbidden {
		t.Fatalf("forged POST status = %d, want 403", rec.Code)
	}
	if got := targetOf(h); got != initialTarget {
		t.Fatalf("/a1 -> %q, want unchanged %q", got, initialTarget)
	}
}

func TestCrossSiteAllowsSameOriginAndNonBrowser(t *testing.T) {
	cases := map[string]map[string]string{
		"same-origin form": {"Cookie": "session=" + sessionValue, "Sec-Fetch-Site": "same-origin"},
		"curl, no headers": {"Cookie": "session=" + sessionValue},
	}
	for name, hdr := range cases {
		t.Run(name, func(t *testing.T) {
			h := newTestVictim(t, Config{Defense: "crosssite"})
			rec := send(h, http.MethodPost, "/settings/target", "url="+url.QueryEscape("https://sho.rt/ok"), hdr)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", rec.Code)
			}
		})
	}
}

// A state change on GET is invisible to both defenses: they only guard unsafe
// methods. This is why GET must never mutate.
func TestGetStateChangeBypassesBothDefenses(t *testing.T) {
	for _, def := range []string{"token", "crosssite"} {
		t.Run(def, func(t *testing.T) {
			h := newTestVictim(t, Config{Defense: def})
			rec := send(h, http.MethodGet, "/legacy/set-target?url="+url.QueryEscape(evilURL), "", map[string]string{
				"Cookie":         "session=" + sessionValue,
				"Sec-Fetch-Site": "cross-site",
				"Sec-Fetch-Mode": "navigate",
			})
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", rec.Code)
			}
			if got := targetOf(h); got != evilURL {
				t.Fatalf("/a1 -> %q, want the attacker's %q", got, evilURL)
			}
		})
	}
}

func TestSessionCookieSameSite(t *testing.T) {
	cases := map[string]http.SameSite{
		"default": 0,
		"lax":     http.SameSiteLaxMode,
		"strict":  http.SameSiteStrictMode,
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			ss, err := sameSiteFor(name)
			if err != nil {
				t.Fatal(err)
			}
			h := newTestVictim(t, Config{SameSite: ss, Defense: "none"})
			cookies := send(h, http.MethodGet, "/login", "", nil).Result().Cookies()
			if len(cookies) != 1 || cookies[0].SameSite != want || !cookies[0].HttpOnly {
				t.Fatalf("cookie = %+v, want SameSite=%v HttpOnly", cookies, want)
			}
		})
	}
}
