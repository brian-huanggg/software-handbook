//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/xss.go` to compile-check only.
package main

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type report struct{ URL, Note string }

var reports []report

// cleanNote strips script tags so reports are safe to show in the admin page.
func cleanNote(s string) string {
	return strings.NewReplacer("<script>", "", "</script>", "").Replace(s)
}

var rowTmpl = template.Must(template.New("row").Parse(`<li>{{.Note}}</li>`))

func admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, "<ul>")
	for _, rep := range reports {
		fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, html.EscapeString(rep.URL), html.EscapeString(rep.URL))
		if err := rowTmpl.Execute(w, map[string]any{"Note": template.HTML(cleanNote(rep.Note))}); err != nil {
			log.Printf("render: %v", err)
		}
	}
	fmt.Fprintln(w, "</ul>")
	if len(reports) > 0 {
		fmt.Fprintf(w, "<script>const lastNote = %q;</script>", reports[len(reports)-1].Note)
	}
}

func main() {
	http.HandleFunc("/admin", admin)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
