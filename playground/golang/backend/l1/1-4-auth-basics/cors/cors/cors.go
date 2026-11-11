// Package cors is a small, allowlist-only CORS middleware for net/http.
package cors

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrBadOrigin reports an allowlist entry that a browser can never send,
// or one that would trust every origin.
var ErrBadOrigin = errors.New("cors: bad allowed origin")

// Policy says which other origins may read this server's responses.
type Policy struct {
	Origins     []string // serialized origins, e.g. "https://app.sho.rt"
	Methods     []string // allowed on preflight, e.g. DELETE
	Headers     []string // request headers the page may set, e.g. Content-Type
	Credentials bool     // allow cookies on cross-origin requests
	MaxAge      time.Duration
}

// New checks the policy once at startup and returns the middleware.
func New(p Policy) (func(http.Handler) http.Handler, error) {
	allowed := make(map[string]bool, len(p.Origins))
	for _, o := range p.Origins {
		if err := checkOrigin(o); err != nil {
			return nil, err
		}
		allowed[o] = true
	}
	methods := strings.Join(p.Methods, ", ")
	headers := strings.Join(p.Headers, ", ")
	maxAge := strconv.Itoa(int(p.MaxAge.Seconds()))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The answer depends on Origin, so a cache must not reuse it
			// for another origin, even when this one is refused.
			w.Header().Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			preflight := r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != ""
			if !allowed[origin] {
				if preflight {
					// No CORS headers: the browser will not send the real request.
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// Not our call to block: same-origin pages and curl send no
				// Origin, and a simple request has already arrived anyway.
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			if p.Credentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if !preflight {
				next.ServeHTTP(w, r)
				return
			}
			// Preflights carry no cookies, so they are answered here,
			// before the auth middleware and the mux can reject them.
			h.Set("Access-Control-Allow-Methods", methods)
			h.Set("Access-Control-Allow-Headers", headers)
			h.Set("Access-Control-Max-Age", maxAge)
			w.WriteHeader(http.StatusNoContent)
		})
	}, nil
}

// checkOrigin accepts only the exact form a browser puts in the Origin
// header: lower-case scheme://host[:port], no path, no default port.
func checkOrigin(o string) error {
	if o == "*" || o == "null" {
		return fmt.Errorf("%w: %q would trust any page", ErrBadOrigin, o)
	}
	u, err := url.Parse(o)
	if err != nil {
		return fmt.Errorf("%w: %q: %v", ErrBadOrigin, o, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("%w: %q needs scheme://host", ErrBadOrigin, o)
	}
	canonical := u.Scheme + "://" + strings.ToLower(u.Host)
	defaultPort := map[string]string{"http": "80", "https": "443"}[u.Scheme]
	if u.Port() == defaultPort {
		canonical = u.Scheme + "://" + strings.ToLower(u.Hostname())
	}
	if o != canonical {
		return fmt.Errorf("%w: %q never matches, browsers send %q", ErrBadOrigin, o, canonical)
	}
	return nil
}
