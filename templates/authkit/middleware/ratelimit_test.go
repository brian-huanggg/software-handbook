package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func TestLimiter_BurstThenRefill(t *testing.T) {
	c := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := middleware.NewLimiter(10*time.Second, 3).WithClock(c.now)

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d within burst was denied", i+1)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait != 10*time.Second {
		t.Fatalf("over burst = %v, %v; want false, 10s", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be independent")
	}
	c.t = c.t.Add(10 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("one token should have refilled")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("only one token should have refilled")
	}
}

func TestLimiter_SweptKeyStartsFull(t *testing.T) {
	c := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := middleware.NewLimiter(time.Second, 2).WithClock(c.now)
	l.Allow("k")
	l.Allow("k")
	c.t = c.t.Add(time.Hour) // refilled, and old enough to sweep
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d after sweep denied", i+1)
		}
	}
}

func TestLimiter_NilAllowsAll(t *testing.T) {
	var l *middleware.Limiter
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("nil limiter must allow")
	}
}

func TestRateLimit_Answers429WithRetryAfter(t *testing.T) {
	l := middleware.NewLimiter(90*time.Second, 1)
	h := middleware.RateLimit(l, middleware.ClientIP)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	call := func(addr string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/token/login", nil)
		r.RemoteAddr = addr
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := call("192.0.2.1:1111"); w.Code != 200 {
		t.Fatalf("first = %d", w.Code)
	}
	w := call("192.0.2.1:2222") // same IP, other port
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "90" {
		t.Fatalf("second = %d, Retry-After %q; want 429, 90", w.Code, w.Header().Get("Retry-After"))
	}
	if w := call("192.0.2.2:1111"); w.Code != 200 {
		t.Fatalf("other IP = %d", w.Code)
	}
}

func TestClientIP(t *testing.T) {
	for addr, want := range map[string]string{
		"192.0.2.1:1234":             "192.0.2.1",
		"[2001:db8:1:2:3::4]:1234":   "2001:db8:1:2::/64",
		"[2001:db8:1:2:ffff::]:5678": "2001:db8:1:2::/64", // same /64, same key
		"[::ffff:192.0.2.1]:1234":    "192.0.2.1",
		"garbage":                    "garbage",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = addr
		if got := middleware.ClientIP(r); got != want {
			t.Errorf("ClientIP(%q) = %q, want %q", addr, got, want)
		}
	}
}
