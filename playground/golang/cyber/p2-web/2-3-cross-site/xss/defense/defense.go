// Package defense holds the server-side XSS defense layers that are not
// "just use html/template": a per-request CSP nonce.
package defense

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

type nonceKey struct{}

// NonceFrom returns the nonce CSP stored in ctx, or "" when CSP is not in use.
func NonceFrom(ctx context.Context) string {
	n, _ := ctx.Value(nonceKey{}).(string)
	return n
}

// Policy is a nonce-based CSP: only scripts carrying the nonce run. Inline
// event handlers and javascript: URLs have no nonce, so they are blocked too.
func Policy(nonce string) string {
	return "script-src 'nonce-" + nonce + "'; object-src 'none'; base-uri 'none'"
}

// CSP gives every response a fresh nonce. A nonce reused across responses is
// just a static allowlist entry the attacker can read and copy.
func CSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		nonce := base64.StdEncoding.EncodeToString(b)
		w.Header().Set("Content-Security-Policy", Policy(nonce))
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nonceKey{}, nonce)))
	})
}
