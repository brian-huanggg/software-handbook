package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sync"

	"handbook/playground/golang/cyber/csrf/defense"
)

// The victim: a v1 short-URL service. alice owns the short link /a1 and can
// change where it points. Login is faked with a fixed cookie.
const (
	sessionValue  = "alice-session"
	initialTarget = "https://sho.rt/docs"
)

// Config picks the cookie attribute and the defense the victim runs with.
type Config struct {
	SameSite http.SameSite // zero value: no SameSite attribute, browser default
	Defense  string        // none, token, crosssite
}

type victim struct {
	mu     sync.Mutex
	target string
	key    []byte
	logger *log.Logger
}

func newVictim(cfg Config, logger *log.Logger) (http.Handler, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("token key: %w", err)
	}
	v := &victim{target: initialTarget, key: key, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", v.login(cfg.SameSite))
	mux.HandleFunc("GET /a1", v.redirect)
	mux.Handle("GET /settings", requireSession(http.HandlerFunc(v.settings)))
	mux.Handle("POST /settings/target", requireSession(http.HandlerFunc(v.setTarget)))
	// Legacy shortcut kept for old bookmarks: changes state on GET.
	mux.Handle("GET /legacy/set-target", requireSession(http.HandlerFunc(v.setTarget)))

	switch cfg.Defense {
	case "none":
		return mux, nil
	case "token":
		return defense.Token(key, sessionOf)(mux), nil
	case "crosssite":
		return defense.CrossSite(mux)
	}
	return nil, fmt.Errorf("unknown defense %q (none, token, crosssite)", cfg.Defense)
}

func sessionOf(r *http.Request) string {
	c, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}

func requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sessionOf(r) != sessionValue {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (v *victim) login(sameSite http.SameSite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "session", Value: sessionValue, Path: "/", MaxAge: 3600,
			HttpOnly: true, SameSite: sameSite,
		})
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := fmt.Fprintln(w, "已用 alice 登入，去 /settings 看看"); err != nil {
			v.logger.Printf("write login: %v", err)
		}
	}
}

func (v *victim) current() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.target
}

func (v *victim) redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, v.current(), http.StatusFound)
}

var settingsTmpl = template.Must(template.New("settings").Parse(`<!doctype html>
<meta charset="utf-8"><title>settings</title>
<p>/a1 現在指向：<b>{{.Target}}</b></p>
<form method="POST" action="/settings/target">
  <input name="url" value="{{.Target}}">
  <input type="hidden" name="csrf_token" value="{{.Token}}">
  <button>更新</button>
</form>
`))

func (v *victim) settings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := settingsTmpl.Execute(w, map[string]string{
		"Target": v.current(),
		"Token":  defense.TokenFor(v.key, sessionOf(r)),
	})
	if err != nil {
		v.logger.Printf("render settings: %v", err)
	}
}

// setTarget serves both the POST form and the legacy GET: FormValue reads the
// body or the query string alike.
func (v *victim) setTarget(w http.ResponseWriter, r *http.Request) {
	raw := r.FormValue("url")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}
	v.mu.Lock()
	v.target = raw
	v.mu.Unlock()
	v.logger.Printf("%s %s Origin=%q Sec-Fetch-Site=%q -> /a1 now %s",
		r.Method, r.URL.Path, r.Header.Get("Origin"), r.Header.Get("Sec-Fetch-Site"), raw)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
