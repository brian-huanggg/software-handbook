package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"slices"
	"sync"
)

// linkStore is the short-URL service's data, kept in memory so the demo has
// no dependencies.
type linkStore struct {
	mu    sync.Mutex
	links map[string]string
}

func newLinkStore() *linkStore {
	return &linkStore{links: map[string]string{
		"go":   "https://go.dev",
		"mdn":  "https://developer.mozilla.org",
		"todo": "https://example.com/my-private-todo",
	}}
}

// newAPI is the short-URL API. It sends no CORS headers on purpose: this
// chapter shows what the browser does before any server opts in.
func newAPI(store *linkStore, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/links", func(w http.ResponseWriter, r *http.Request) {
		// Logging Origin proves the request reached us even when the
		// browser later hides the response from the calling page.
		logger.Printf("%s %s Origin=%q", r.Method, r.URL.Path, r.Header.Get("Origin"))
		store.mu.Lock()
		codes := make([]string, 0, len(store.links))
		for code := range store.links {
			codes = append(codes, code)
		}
		store.mu.Unlock()
		slices.Sort(codes)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(codes); err != nil {
			logger.Printf("encode links: %v", err)
		}
	})
	mux.HandleFunc("POST /api/links/delete", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s Origin=%q", r.Method, r.URL.Path, r.Header.Get("Origin"))
		code := r.FormValue("code")
		store.mu.Lock()
		delete(store.links, code)
		store.mu.Unlock()
		logger.Printf("deleted %q", code)
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<meta charset="utf-8">
<title>另一個 origin 的網頁</title>
<button id="read">讀取短網址清單</button>
<button id="del">刪除 todo</button>
<pre id="out"></pre>
<script>
const api = {{.}};
const out = document.getElementById("out");
document.getElementById("read").onclick = async () => {
  try {
    const res = await fetch(api + "/api/links");
    out.textContent = "讀到了：" + await res.text();
  } catch (err) {
    out.textContent = "讀不到回應：" + err;
  }
};
document.getElementById("del").onclick = async () => {
  try {
    await fetch(api + "/api/links/delete", {
      method: "POST",
      headers: {"Content-Type": "application/x-www-form-urlencoded"},
      body: "code=todo",
    });
    out.textContent = "刪除請求成功";
  } catch (err) {
    out.textContent = "讀不到回應：" + err;
  }
};
</script>
`))

// newPage serves a page from a second origin that calls the API with fetch.
// The delete button sends a form-encoded POST, a "simple" request that the
// browser sends without asking the server first (backend 1.4 covers the
// other kind). The comment lives here because html/template strips JS comments.
func newPage(apiOrigin string, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pageTmpl.Execute(w, apiOrigin); err != nil {
			logger.Printf("render page: %v", err)
		}
	})
}
