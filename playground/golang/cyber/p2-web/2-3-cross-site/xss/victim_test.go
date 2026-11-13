package main

import (
	"html"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

const payload = `<img src=x onerror="alert(1)">`

func newTestVictim(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	h, err := newVictim(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

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

func aliceCookie() map[string]string { return map[string]string{"Cookie": "session=" + sessionValue} }

func fileReport(h http.Handler, u, note string) *httptest.ResponseRecorder {
	form := url.Values{"url": {u}, "note": {note}}
	return send(h, http.MethodPost, "/reports", form.Encode(), nil)
}

func TestReflectedRunsWithoutDefense(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "none"})
	rec := send(h, http.MethodGet, "/search?q="+url.QueryEscape(payload), "", nil)
	if !strings.Contains(rec.Body.String(), payload) {
		t.Fatalf("payload should come back as live markup, got:\n%s", rec.Body)
	}
}

func TestEncodeNeutralisesReflected(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "encode"})
	body := send(h, http.MethodGet, "/search?q="+url.QueryEscape(payload), "", nil).Body.String()
	if strings.Contains(body, "<img") {
		t.Fatalf("payload became a live tag:\n%s", body)
	}
	if !strings.Contains(body, "&lt;img") {
		t.Fatalf("payload should be shown as text, got:\n%s", body)
	}
}

func TestStoredReachesAdminWithoutDefense(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "none"})
	if rec := fileReport(h, "https://example.com", payload); rec.Code != http.StatusCreated {
		t.Fatalf("report status = %d, want 201", rec.Code)
	}
	body := send(h, http.MethodGet, "/admin", "", aliceCookie()).Body.String()
	if !strings.Contains(body, payload) {
		t.Fatalf("stored payload should reach alice as markup, got:\n%s", body)
	}
}

func TestEncodeNeutralisesStored(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "encode"})
	if rec := fileReport(h, "https://example.com", payload); rec.Code != http.StatusCreated {
		t.Fatalf("report status = %d, want 201", rec.Code)
	}
	body := send(h, http.MethodGet, "/admin", "", aliceCookie()).Body.String()
	if strings.Contains(body, "<img") {
		t.Fatalf("stored payload became a live tag:\n%s", body)
	}
}

func TestJavascriptURLInHref(t *testing.T) {
	const js = "javascript:alert(1)"
	none := newTestVictim(t, Config{Defense: "none"})
	fileReport(none, js, "x")
	if body := send(none, http.MethodGet, "/admin", "", aliceCookie()).Body.String(); !strings.Contains(body, `href="javascript:`) {
		t.Fatalf("javascript: scheme should survive into href (the template only re-encodes the rest), got:\n%s", body)
	}

	enc := newTestVictim(t, Config{Defense: "encode"})
	if rec := fileReport(enc, js, "x"); rec.Code != http.StatusBadRequest {
		t.Fatalf("encode mode report status = %d, want 400 for a non-http(s) scheme", rec.Code)
	}
}

func TestPreviewSink(t *testing.T) {
	cases := map[string]struct{ want, avoid string }{
		"none":   {"innerHTML", "textContent"},
		"encode": {"textContent", "innerHTML"},
	}
	for def, c := range cases {
		t.Run(def, func(t *testing.T) {
			body := send(newTestVictim(t, Config{Defense: def}), http.MethodGet, "/preview", "", nil).Body.String()
			if !strings.Contains(body, c.want) || strings.Contains(body, c.avoid) {
				t.Fatalf("want sink %s and not %s, got:\n%s", c.want, c.avoid, body)
			}
		})
	}
}

var nonceRe = regexp.MustCompile(`nonce="([^"]+)"`)

func TestCSPNonceMatchesPageAndChangesPerRequest(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "csp"})
	seen := map[string]bool{}
	for range 2 {
		rec := send(h, http.MethodGet, "/preview", "", nil)
		csp := rec.Header().Get("Content-Security-Policy")
		m := nonceRe.FindStringSubmatch(rec.Body.String())
		if m == nil {
			t.Fatalf("page has no nonce attribute:\n%s", rec.Body)
		}
		nonce := html.UnescapeString(m[1]) // the browser decodes attribute entities before matching
		if !strings.Contains(csp, "'nonce-"+nonce+"'") {
			t.Fatalf("header %q does not carry the page nonce %q", csp, nonce)
		}
		if strings.Contains(csp, "unsafe-inline") {
			t.Fatalf("policy must not allow unsafe-inline: %q", csp)
		}
		seen[nonce] = true
	}
	if len(seen) != 2 {
		t.Fatal("nonce was reused across responses")
	}
}

func TestCSPIsNotTheFix(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "csp"})
	body := send(h, http.MethodGet, "/search?q="+url.QueryEscape(payload), "", nil).Body.String()
	if !strings.Contains(body, payload) {
		t.Fatal("csp mode still injects the payload into the page; CSP only stops the browser from running it")
	}
}

func TestNoCSPHeaderOtherwise(t *testing.T) {
	for _, def := range []string{"none", "encode"} {
		rec := send(newTestVictim(t, Config{Defense: def}), http.MethodGet, "/preview", "", nil)
		if got := rec.Header().Get("Content-Security-Policy"); got != "" {
			t.Fatalf("%s mode sent CSP %q", def, got)
		}
	}
}

func TestSessionCookieHttpOnly(t *testing.T) {
	for _, on := range []bool{false, true} {
		rec := send(newTestVictim(t, Config{Defense: "none", HttpOnly: on}), http.MethodGet, "/login", "", nil)
		cs := rec.Result().Cookies()
		if len(cs) != 1 || cs[0].HttpOnly != on {
			t.Fatalf("HttpOnly = %v, want %v (cookies: %v)", cs, on, cs)
		}
	}
}

// HttpOnly hides the cookie from document.cookie, but the injected script's own
// request carries the cookie anyway. The server cannot tell it from alice.
func TestHttpOnlyDoesNotStopRiding(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "none", HttpOnly: true})
	hdr := aliceCookie()
	hdr["Sec-Fetch-Site"] = "same-origin" // what fetch() from an XSS-ed page sends
	body := "url=" + url.QueryEscape("http://evil.test:9666/phish")
	if rec := send(h, http.MethodPost, "/settings/target", body, hdr); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := send(h, http.MethodGet, "/a1", "", nil).Header().Get("Location"); got != "http://evil.test:9666/phish" {
		t.Fatalf("/a1 -> %q, want the attacker's target", got)
	}
}
