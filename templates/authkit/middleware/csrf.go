package middleware

import (
	"net/http"

	"github.com/brian-huanggg/software-handbook/templates/authkit/respond"
)

const (
	// CSRFHeader must be present on every state-changing request.
	CSRFHeader = "X-Requested-With"
	CSRFValue  = "XMLHttpRequest"
)

// CSRF protects cookie-authenticated endpoints against cross-site request
// forgery with two independent checks on unsafe methods (anything but
// GET, HEAD, OPTIONS):
//
//  1. A custom header is required. A cross-site page cannot add one without
//     a CORS preflight, and CORS only approves the listed origins. HTML forms
//     cannot set it at all.
//  2. If the browser sent an Origin, it must be an allowed origin. There is
//     no "same host as the API" fallback: behind a proxy r.Host is often the
//     internal host, and it ignores the scheme. A page served from the API's
//     own origin must be listed like any other.
//
// Safe methods must never change state, since they are not checked.
func CSRF(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := originSet(allowedOrigins)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && !allowed[origin] {
				respond.Error(w, http.StatusForbidden, "forbidden origin")
				return
			}
			if r.Header.Get(CSRFHeader) != CSRFValue {
				respond.Error(w, http.StatusForbidden, "missing "+CSRFHeader+" header")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
