// Command upload-with-data serves POST /links with a preview image attached.
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	store := DirStore{Dir: "data"}
	if err := store.init(); err != nil {
		log.Fatal(err)
	}
	h := &Handler{
		Store:    store,
		Repo:     &MemRepo{},
		MaxBody:  3 << 20,
		MaxFile:  2 << 20,
		MaxField: 4 << 10,
	}
	mux := http.NewServeMux()
	mux.Handle("POST /links", h)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second, // bounds a slow upload
	}
	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
