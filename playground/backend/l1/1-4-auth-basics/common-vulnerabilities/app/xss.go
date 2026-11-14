package app

import (
	"fmt"
	"html/template"
	"net/http"
)

func reportUnsafe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><p>備註：%s</p>", r.URL.Query().Get("note"))
}

var reportTmpl = template.Must(template.New("report").
	Parse("<!doctype html><p>備註：{{.}}</p>"))

func reportSafe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// A second line of defense: even if escaping is missed somewhere,
	// the browser refuses to run any script on this page.
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	if err := reportTmpl.Execute(w, r.URL.Query().Get("note")); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
