// 短網址服務 v2 的示範：用旗標在舊、新兩條跳轉路徑之間切換。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	flags "handbook/playground/golang/backend/feature-flags"
)

const flagNewLookup = "new-lookup-path"

func main() {
	var store flags.Store

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	path := "flags.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	go store.Watch(ctx, path, time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		user := r.Header.Get("X-User-ID") // 示範用；正式環境從驗證過的 session 取
		if store.Enabled(flagNewLookup, user) {
			fmt.Fprintf(w, "new path: %s\n", r.PathValue("code"))
			return
		}
		fmt.Fprintf(w, "old path: %s\n", r.PathValue("code"))
	})

	srv := &http.Server{Addr: "127.0.0.1:8080", Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
