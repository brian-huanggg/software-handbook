package app

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newServer(t *testing.T) (*Server, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	return NewServer(slog.New(slog.NewTextHandler(&logs, nil))), &logs
}

func get(t *testing.T, h http.Handler, target string) (int, string, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	return rec.Code, string(body), rec.Result().Header
}

func TestReflectedXSS(t *testing.T) {
	s, _ := newServer(t)
	payload := `<img src=x onerror=alert(1)>`
	target := "/report?note=" + url.QueryEscape(payload)

	_, body, _ := get(t, s.Vulnerable(), target)
	if !strings.Contains(body, payload) {
		t.Errorf("vulnerable page should echo the payload raw, got %q", body)
	}

	_, body, hdr := get(t, s.Safe(), target)
	if strings.Contains(body, "<img") {
		t.Errorf("safe page must escape the payload, got %q", body)
	}
	if !strings.Contains(body, "&lt;img") {
		t.Errorf("safe page should show the payload as text, got %q", body)
	}
	if hdr.Get("Content-Security-Policy") == "" {
		t.Error("safe page should send a CSP header")
	}
}

// crossSiteDelete is what a form on another site makes the victim's browser send.
func crossSiteDelete(h http.Handler) int {
	req := httptest.NewRequest(http.MethodPost, "/links/delete",
		strings.NewReader("slug=go"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://evil.test")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: "sid", Value: sessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestCSRFDelete(t *testing.T) {
	s, _ := newServer(t)
	if code := crossSiteDelete(s.Vulnerable()); code != http.StatusNoContent {
		t.Fatalf("vulnerable: status = %d, want 204", code)
	}
	if s.HasLink("go") {
		t.Error("vulnerable: the cross-site request should have deleted the link")
	}

	s, _ = newServer(t)
	if code := crossSiteDelete(s.Safe()); code != http.StatusForbidden {
		t.Fatalf("safe: status = %d, want 403", code)
	}
	if !s.HasLink("go") {
		t.Error("safe: the link must survive a cross-site request")
	}
}

func TestSameOriginDeleteStillWorks(t *testing.T) {
	s, _ := newServer(t)
	req := httptest.NewRequest(http.MethodPost, "/links/delete",
		strings.NewReader("slug=go"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: "sid", Value: sessionID})
	rec := httptest.NewRecorder()
	s.Safe().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestMeExposure(t *testing.T) {
	s, _ := newServer(t)
	_, body, _ := get(t, s.Vulnerable(), "/me")
	if !strings.Contains(body, "argon2id") {
		t.Errorf("vulnerable /me should leak the hash, got %q", body)
	}
	_, body, _ = get(t, s.Safe(), "/me")
	if strings.Contains(body, "argon2id") || strings.Contains(body, "password") {
		t.Errorf("safe /me must not include the hash, got %q", body)
	}
	if !strings.Contains(body, "alice@example.com") {
		t.Errorf("safe /me should still include the email, got %q", body)
	}
}

func TestErrorDetailExposure(t *testing.T) {
	s, logs := newServer(t)
	_, body, _ := get(t, s.Vulnerable(), "/links/lookup?slug=go")
	if !strings.Contains(body, "10.0.3.7") {
		t.Errorf("vulnerable error should leak the address, got %q", body)
	}
	_, body, _ = get(t, s.Safe(), "/links/lookup?slug=go")
	if strings.Contains(body, "10.0.3.7") || strings.Contains(body, "SELECT") {
		t.Errorf("safe error must be generic, got %q", body)
	}
	if !strings.Contains(logs.String(), "10.0.3.7") {
		t.Error("the detail should still reach the server log")
	}
}
