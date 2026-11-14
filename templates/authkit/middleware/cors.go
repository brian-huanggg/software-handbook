package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ValidateOrigin checks that o is a bare origin: scheme://host[:port], no
// path, no trailing slash, and not the wildcard "*".
func ValidateOrigin(o string) error {
	u, err := url.Parse(o)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || strings.HasSuffix(o, "/") {
		return fmt.Errorf("invalid origin %q: want scheme://host[:port], e.g. http://localhost:3000", o)
	}
	return nil
}

func originSet(origins []string) map[string]bool {
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		if err := ValidateOrigin(o); err != nil {
			panic("middleware: " + err.Error())
		}
		set[o] = true
	}
	return set
}

// CORS lets browsers on the listed origins call the API with credentials.
//
// Origins must be listed explicitly: a wildcard cannot be combined with
// cookies, and echoing arbitrary origins would be just as bad, so anything
// not in the list gets no CORS headers at all. An empty list disables CORS.
// It panics on malformed origins, like a bad route pattern.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := originSet(origins)
	return func(next http.Handler) http.Handler {
		if len(allowed) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Add("Vary", "Origin")
			preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""

			if !allowed[origin] {
				if preflight {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r) // served, but the browser will not expose it
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			if preflight {
				h.Add("Vary", "Access-Control-Request-Method")
				h.Add("Vary", "Access-Control-Request-Headers")
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
