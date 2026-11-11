package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"handbook/playground/golang/backend/cors/cors"
)

// sessionCookie stands in for real login (1.4 Session 與 Cookie).
const sessionCookie = "demo-session"

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

// requireSession rejects requests without the login cookie.
func requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err != nil || c.Value != sessionCookie {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newMux(store *linkStore, logger *log.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/links", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("DELETE /api/links/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		logger.Printf("%s %s Origin=%q", r.Method, r.URL.Path, r.Header.Get("Origin"))
		store.mu.Lock()
		delete(store.links, code)
		store.mu.Unlock()
		logger.Printf("deleted %q", code)
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// newAPI wires the layers. Order matters: CORS must be outermost so that
// preflights, which never carry cookies, are answered before requireSession
// returns 401 and before the mux returns 405 for OPTIONS.
func newAPI(store *linkStore, logger *log.Logger, allowed []string) (http.Handler, error) {
	withCORS, err := cors.New(cors.Policy{
		Origins:     allowed,
		Methods:     []string{"GET", "DELETE"},
		Headers:     []string{"Content-Type"},
		Credentials: true,
		MaxAge:      10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	api := withCORS(requireSession(newMux(store, logger)))

	root := http.NewServeMux()
	root.Handle("/api/", api)
	root.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "session", Value: sessionCookie, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte("已登入，可以關掉這個分頁\n")); err != nil {
			logger.Printf("write login: %v", err)
		}
	})
	return root, nil
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<meta charset="utf-8">
<title>{{.Name}}</title>
<h1>{{.Name}}</h1>
<button id="read">讀取短網址清單</button>
<button id="del">刪除 todo</button>
<pre id="out"></pre>
<script>
const api = {{.API}};
const out = document.getElementById("out");
document.getElementById("read").onclick = async () => {
  try {
    const res = await fetch(api + "/api/links", {credentials: "include"});
    out.textContent = res.status + " 讀到了：" + await res.text();
  } catch (err) {
    out.textContent = "讀不到回應：" + err;
  }
};
document.getElementById("del").onclick = async () => {
  try {
    const res = await fetch(api + "/api/links/todo", {method: "DELETE", credentials: "include"});
    out.textContent = "刪除回應：" + res.status;
  } catch (err) {
    out.textContent = "讀不到回應：" + err;
  }
};
</script>
`))

// newPage serves a front end from its own origin. DELETE is not a simple
// method, so the browser preflights it; the comment lives here because
// html/template strips JS comments.
func newPage(name, apiOrigin string, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Name, API string }{name, apiOrigin}
		if err := pageTmpl.Execute(w, data); err != nil {
			logger.Printf("render page: %v", err)
		}
	})
}
