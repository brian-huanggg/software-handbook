// Command authn-vs-authz serves a tiny link API that separates
// authentication (who are you?) from authorization (may you do this?).
package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	srv := &Server{
		store: NewStore(),
		auth: &Authenticator{users: map[string]User{
			"alice-token": {ID: "alice", Role: RoleUser},
			"bob-token":   {ID: "bob", Role: RoleUser},
			"root-token":  {ID: "root", Role: RoleAdmin},
		}},
	}
	hs := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("listening on", hs.Addr)
	log.Fatal(hs.ListenAndServe())
}
