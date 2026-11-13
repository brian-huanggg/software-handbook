// Package policy holds origin checks a CORS middleware might use: four broken
// ones that appear in real-world findings, and the fix.
package policy

import (
	"net/http"
	"strings"
)

// Check reports whether a page from origin may read responses that were
// fetched with the user's cookies.
type Check func(origin string) bool

// Reflect trusts whatever Origin the request claims. Vulnerable.
func Reflect() Check {
	return func(origin string) bool { return origin != "" }
}

// AllowNull is an exact allowlist that also trusts "null", usually added so
// pages opened from file: work in development. Vulnerable: any site can send
// Origin: null from a sandboxed iframe.
func AllowNull(origins ...string) Check {
	exact := Allowlist(origins...)
	return func(origin string) bool { return origin == "null" || exact(origin) }
}

// Suffix trusts any origin ending in domain. Vulnerable: https://evilsho.rt
// also ends in "sho.rt".
func Suffix(domain string) Check {
	return func(origin string) bool { return strings.HasSuffix(origin, domain) }
}

// Prefix trusts any origin starting with prefix. Vulnerable:
// https://app.sho.rt.evil.example also starts with "https://app.sho.rt".
func Prefix(prefix string) Check {
	return func(origin string) bool { return strings.HasPrefix(origin, prefix) }
}

// Allowlist trusts only origins that match an entry exactly. The fix.
// Entries must be canonical origins (see backend 1.4, checkOrigin).
func Allowlist(origins ...string) Check {
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		set[o] = true
	}
	return func(origin string) bool { return set[origin] }
}

// CORS grants credentialed reads to origins that pass check.
func CORS(check Check) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Origin")
			if origin := r.Header.Get("Origin"); check(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			next.ServeHTTP(w, r)
		})
	}
}
