package middleware

import (
	"net/http"
	"net/url"
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
//  2. If the browser sent an Origin, it must be an allowed origin or the
//     API's own host.
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
			if origin := r.Header.Get("Origin"); origin != "" && !allowed[origin] && !sameHost(origin, r.Host) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
			if r.Header.Get(CSRFHeader) != CSRFValue {
				http.Error(w, "missing "+CSRFHeader+" header", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sameHost(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && u.Host == host
}
