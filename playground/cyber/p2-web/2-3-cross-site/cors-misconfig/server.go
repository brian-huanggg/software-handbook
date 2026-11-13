// Command cors-misconfig runs a deliberately misconfigured short-URL API
// alongside a page that reads from it cross-origin, all on 127.0.0.1. It is a
// self-contained lab for backend 1.4's CORS lesson: pick a -mode and see which
// origin checks leak a logged-in user's data to another origin.
//
// Everything here targets only your own machine. Do not point it at anything else.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"handbook/playground/cyber/cors-misconfig/policy"
)

// The victim API: a v1 short-URL service where GET /api/me returns the
// logged-in user's private API key. Login is faked with a fixed cookie.
const (
	sessionValue = "alice-session"
	apiKey       = "sk-demo-alice-7f3a" // fake value; stands in for anything private
	appOrigin    = "https://app.sho.rt" // the one front end that should be trusted
)

// checkFor maps the -mode flag to one of policy's origin checks.
func checkFor(mode string) (policy.Check, error) {
	switch mode {
	case "reflect":
		return policy.Reflect(), nil
	case "null":
		return policy.AllowNull(appOrigin), nil
	case "suffix":
		return policy.Suffix("sho.rt"), nil
	case "prefix":
		return policy.Prefix(appOrigin), nil
	case "fixed":
		return policy.Allowlist(appOrigin), nil
	}
	return nil, fmt.Errorf("unknown mode %q (reflect, null, suffix, prefix, fixed)", mode)
}

func requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err != nil || c.Value != sessionValue {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newAPI(check policy.Check, logger *log.Logger) http.Handler {
	me := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("GET /api/me Origin=%q -> Allow-Origin=%q",
			r.Header.Get("Origin"), w.Header().Get("Access-Control-Allow-Origin"))
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"user": "alice", "api_key": apiKey}); err != nil {
			logger.Printf("encode me: %v", err)
		}
	})
	mux := http.NewServeMux()
	mux.Handle("GET /api/me", policy.CORS(check)(requireSession(me)))
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "session", Value: sessionValue, Path: "/", MaxAge: 3600,
			HttpOnly: true, SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := fmt.Fprintln(w, "已用 alice 登入，可以關掉這個分頁"); err != nil {
			logger.Printf("write login: %v", err)
		}
	})
	return mux
}

const (
	apiAddr    = "127.0.0.1:8080" // the victim API
	attackAddr = "127.0.0.1:9666" // a page on an origin the API never meant to trust
)

func main() {
	mode := flag.String("mode", "reflect", "origin check: reflect, null, suffix, prefix, fixed")
	flag.Parse()
	check, err := checkFor(*mode)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.Default()

	servers := []*http.Server{
		{Addr: apiAddr, Handler: newAPI(check, logger), ReadHeaderTimeout: 5 * time.Second},
		{Addr: attackAddr, Handler: attackPage(logger), ReadHeaderTimeout: 5 * time.Second},
	}
	errc := make(chan error, len(servers))
	for _, s := range servers {
		go func() { errc <- s.ListenAndServe() }()
	}
	logger.Printf("mode=%s", *mode)
	logger.Printf("1. 先登入：http://%s/login", apiAddr)
	logger.Printf("2. 再打開另一個 origin 的頁面：http://%s/", attackAddr)
	log.Fatal(<-errc)
}
