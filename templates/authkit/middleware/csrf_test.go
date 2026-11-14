package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
)

func csrfServe(method string, hdr map[string]string) (int, bool) {
	called := false
	h := middleware.CSRF([]string{allowed})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	r := httptest.NewRequest(method, "http://api.test/x", nil)
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code, called
}

func TestCSRF(t *testing.T) {
	good := map[string]string{"X-Requested-With": "XMLHttpRequest"}
	with := func(k, v string) map[string]string {
		m := map[string]string{"X-Requested-With": "XMLHttpRequest"}
		m[k] = v
		return m
	}
	for name, c := range map[string]struct {
		method string
		hdr    map[string]string
		allow  bool
	}{
		"POST with header":               {"POST", good, true},
		"POST without header":            {"POST", nil, false},
		"POST wrong header value":        {"POST", map[string]string{"X-Requested-With": "x"}, false},
		"PUT without header":             {"PUT", nil, false},
		"DELETE without header":          {"DELETE", nil, false},
		"PATCH without header":           {"PATCH", nil, false},
		"POST header + allowed origin":   {"POST", with("Origin", allowed), true},
		"POST header + same-host origin": {"POST", with("Origin", "http://api.test"), false},
		"POST header + other scheme":     {"POST", with("Origin", "https://localhost:3000"), false},
		"POST header + evil origin":      {"POST", with("Origin", "http://evil.example"), false},
		"POST header + null origin":      {"POST", with("Origin", "null"), false},
		"POST evil origin, no header":    {"POST", map[string]string{"Origin": "http://evil.example"}, false},
		"GET without header":             {"GET", nil, true},
		"HEAD without header":            {"HEAD", nil, true},
		"OPTIONS without header":         {"OPTIONS", nil, true},
	} {
		code, called := csrfServe(c.method, c.hdr)
		if called != c.allow || (c.allow && code != 200) || (!c.allow && code != http.StatusForbidden) {
			t.Errorf("%s: code=%d called=%v, want allow=%v", name, code, called, c.allow)
		}
	}
}
