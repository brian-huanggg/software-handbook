package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sync"

	"handbook/playground/cyber/xss/defense"
)

// The victim: a v1 short-URL service. Anyone can report a link, alice is the
// admin who reads the reports, and login is faked with a fixed cookie.
const (
	sessionValue  = "alice-session"
	initialTarget = "https://sho.rt/docs"
)

// Config picks the defense the victim runs with.
type Config struct {
	Defense  string // none, encode, csp
	HttpOnly bool   // session cookie HttpOnly
}

type report struct{ URL, Note string }

type victim struct {
	cfg     Config
	mu      sync.Mutex
	target  string
	reports []report
	logger  *log.Logger
}

func newVictim(cfg Config, logger *log.Logger) (http.Handler, error) {
	v := &victim{cfg: cfg, target: initialTarget, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", v.login)
	mux.HandleFunc("GET /a1", v.redirect)
	mux.HandleFunc("GET /search", v.search)
	mux.HandleFunc("GET /preview", v.preview)
	mux.HandleFunc("POST /reports", v.addReport)
	mux.Handle("GET /admin", requireSession(http.HandlerFunc(v.admin)))
	mux.Handle("POST /settings/target", requireSession(http.HandlerFunc(v.setTarget)))

	switch cfg.Defense {
	case "none", "encode":
		return mux, nil
	case "csp":
		return defense.CSP(mux), nil
	}
	return nil, fmt.Errorf("unknown defense %q (none, encode, csp)", cfg.Defense)
}

// safe reports whether user input is encoded for its output context.
func (v *victim) safe() bool { return v.cfg.Defense == "encode" }

// text is how user input reaches an HTML body. html/template escapes a plain
// string; template.HTML tells it "this is already markup, do not touch it",
// which is exactly the opt-out that creates the bug.
func (v *victim) text(s string) any {
	if v.safe() {
		return s
	}
	return template.HTML(s)
}

// href does the same for a URL attribute: template.URL skips the scheme filter.
func (v *victim) href(s string) any {
	if v.safe() {
		return s
	}
	return template.URL(s)
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

func (v *victim) login(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: sessionValue, Path: "/", MaxAge: 3600,
		HttpOnly: v.cfg.HttpOnly, SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := fmt.Fprintln(w, "已用 alice 登入，去 /admin 看看"); err != nil {
		v.logger.Printf("write login: %v", err)
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

func (v *victim) render(w http.ResponseWriter, r *http.Request, t *template.Template, data map[string]any) {
	data["Nonce"] = defense.NonceFrom(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		v.logger.Printf("render %s: %v", t.Name(), err)
	}
}

var searchTmpl = template.Must(template.New("search").Parse(`<!doctype html>
<meta charset="utf-8"><title>search</title>
<p>找不到符合「{{.Q}}」的短連結</p>
`))

// search echoes the query back: reflected XSS when Q is treated as markup.
func (v *victim) search(w http.ResponseWriter, r *http.Request) {
	v.render(w, r, searchTmpl, map[string]any{"Q": v.text(r.URL.Query().Get("q"))})
}

var adminTmpl = template.Must(template.New("admin").Parse(`<!doctype html>
<meta charset="utf-8"><title>admin</title>
<p>/a1 現在指向：<b>{{.Target}}</b></p>
<ul>{{range .Reports}}
<li><a href="{{.URL}}">檢舉的連結</a>：{{.Note}}</li>{{end}}
</ul>
`))

// addReport stores whatever any visitor sends. alice reads it later in /admin:
// stored XSS when the note is treated as markup.
func (v *victim) addReport(w http.ResponseWriter, r *http.Request) {
	raw, note := r.FormValue("url"), r.FormValue("note")
	if v.safe() {
		// Input validation is the second layer, not the fix: it keeps
		// javascript: out of the data. Output encoding still does the real work.
		if u, err := url.Parse(raw); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
	}
	v.mu.Lock()
	v.reports = append(v.reports, report{URL: raw, Note: note})
	v.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func (v *victim) admin(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock()
	rows := make([]map[string]any, 0, len(v.reports))
	for _, rep := range v.reports {
		rows = append(rows, map[string]any{"URL": v.href(rep.URL), "Note": v.text(rep.Note)})
	}
	v.mu.Unlock()
	v.render(w, r, adminTmpl, map[string]any{"Target": v.current(), "Reports": rows})
}

// setTarget is what alice can do and an attacker's script, running as alice, can too.
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
	v.logger.Printf("%s %s Sec-Fetch-Site=%q -> /a1 now %s", r.Method, r.URL.Path, r.Header.Get("Sec-Fetch-Site"), raw)
	w.WriteHeader(http.StatusNoContent)
}

var previewTmpl = template.Must(template.New("preview").Parse(`<!doctype html>
<meta charset="utf-8"><title>preview</title>
<p>預覽：<span id="out"></span></p>
<script nonce="{{.Nonce}}">
const text = decodeURIComponent(location.hash.slice(1));
const out = document.getElementById("out");
{{if .Safe}}out.textContent = text;{{else}}out.innerHTML = text;{{end}}
</script>
`))

// preview never sends the fragment to the server: the bug is entirely in the
// client-side sink, so the server only decides which sink the page uses.
func (v *victim) preview(w http.ResponseWriter, r *http.Request) {
	v.render(w, r, previewTmpl, map[string]any{"Safe": v.safe()})
}
