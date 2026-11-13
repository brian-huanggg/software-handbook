// Package defense holds the server-side CSRF checks the lab compares: a
// session-bound token, and the standard library's Fetch Metadata / Origin check.
package defense

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

const (
	FormField = "csrf_token"
	Header    = "X-CSRF-Token"
)

// TokenFor derives the anti-CSRF token for one session. Binding it to the
// session (instead of a free-standing random cookie) means an attacker who can
// plant a cookie still cannot produce a matching pair.
func TokenFor(key []byte, session string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(session))
	return hex.EncodeToString(m.Sum(nil))
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// Token rejects unsafe requests that do not carry the token for their session.
func Token(key []byte, session func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if safeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			got := r.Header.Get(Header)
			if got == "" {
				got = r.PostFormValue(FormField)
			}
			want := TokenFor(key, session(r))
			if !hmac.Equal([]byte(got), []byte(want)) {
				http.Error(w, "csrf token mismatch", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CrossSite wraps h with net/http's CrossOriginProtection: for unsafe methods
// it rejects Sec-Fetch-Site values other than same-origin/none, and falls back
// to comparing Origin with Host when the browser sent no Fetch Metadata.
func CrossSite(h http.Handler, trusted ...string) (http.Handler, error) {
	cop := http.NewCrossOriginProtection()
	for _, o := range trusted {
		if err := cop.AddTrustedOrigin(o); err != nil {
			return nil, err
		}
	}
	return cop.Handler(h), nil
}
