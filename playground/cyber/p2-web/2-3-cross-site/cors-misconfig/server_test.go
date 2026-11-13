package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"handbook/playground/cyber/cors-misconfig/policy"
)

const attacker = "http://127.0.0.1:9666"

// get sends a logged-in GET /api/me from origin and returns the response.
func get(t *testing.T, check policy.Check, origin string) *http.Response {
	t.Helper()
	api := newAPI(check, discardLogger())
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/me", nil)
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionValue})
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

// leaks reports whether a browser would hand the body to a page on origin:
// the key is present AND the response authorises that exact origin with
// credentials. This is what the browser checks before releasing the body.
func leaks(res *http.Response, origin string) bool {
	body := readBody(res)
	allow := res.Header.Get("Access-Control-Allow-Origin")
	creds := res.Header.Get("Access-Control-Allow-Credentials") == "true"
	return strings.Contains(body, apiKey) && allow == origin && creds
}

// Each broken mode leaks alice's key, but to a different attacker origin.
// reflect trusts the localhost page directly; null needs a sandboxed iframe
// (Origin: null); suffix and prefix only match a domain the attacker must
// register, so they do not fire for the :9666 page — a point the article makes.
func TestMisconfigsLeak(t *testing.T) {
	cases := map[string]string{
		"reflect": attacker,
		"null":    "null",
		"suffix":  "https://evilsho.rt",
		"prefix":  "https://app.sho.rt.evil.example",
	}
	for mode, origin := range cases {
		check, err := checkFor(mode)
		if err != nil {
			t.Fatal(err)
		}
		if !leaks(get(t, check, origin), origin) {
			t.Errorf("mode %s: expected origin %s to read the key", mode, origin)
		}
	}
}

// The fix: an exact allowlist authorises only the real front end, so the
// attacker's page and a sandboxed iframe both come back without the headers
// the browser needs, even though the handler still ran and returned the key.
func TestFixedModeBlocksAttacker(t *testing.T) {
	check, _ := checkFor("fixed")
	for _, origin := range []string{attacker, "null", "https://evilsho.rt"} {
		if leaks(get(t, check, origin), origin) {
			t.Errorf("mode fixed: origin %s should not be able to read the key", origin)
		}
	}
	// The legitimate front end still works.
	if !leaks(get(t, check, appOrigin), appOrigin) {
		t.Errorf("mode fixed: the real front end %s should still read the key", appOrigin)
	}
}

// requireSession, not CORS, is what stops a non-browser client. curl sends no
// Origin and ignores the headers, so it reads the key under every mode. CORS
// is not access control.
func TestNonBrowserClientAlwaysReads(t *testing.T) {
	check, _ := checkFor("fixed")
	api := newAPI(check, discardLogger())
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionValue})
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	if !strings.Contains(readBody(rec.Result()), apiKey) {
		t.Error("curl-style client with a valid cookie should read the key regardless of CORS")
	}
}

// Without the cookie, there is nothing to steal: the handler never runs.
func TestNoSessionNothingToSteal(t *testing.T) {
	check, _ := checkFor("reflect")
	api := newAPI(check, discardLogger())
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/me", nil)
	req.Header.Set("Origin", attacker)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	if strings.Contains(readBody(rec.Result()), apiKey) {
		t.Error("without a session cookie the key must not be returned")
	}
}
