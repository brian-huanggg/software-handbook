// Command cors runs the short-URL API with a CORS allowlist, plus two
// front ends on other ports: one allowed, one not.
package main

import (
	"log"
	"net/http"
	"time"
)

const (
	apiAddr   = "127.0.0.1:8080"
	appAddr   = "127.0.0.1:9090" // on the allowlist
	otherAddr = "127.0.0.1:9191" // not on the allowlist
)

func main() {
	logger := log.Default()
	api, err := newAPI(newLinkStore(), logger, []string{"http://" + appAddr})
	if err != nil {
		logger.Fatal(err)
	}
	apiOrigin := "http://" + apiAddr
	servers := []*http.Server{
		{Addr: apiAddr, Handler: api, ReadHeaderTimeout: 5 * time.Second},
		{Addr: appAddr, Handler: newPage("管理後台（允許）", apiOrigin, logger), ReadHeaderTimeout: 5 * time.Second},
		{Addr: otherAddr, Handler: newPage("別人的網頁（不允許）", apiOrigin, logger), ReadHeaderTimeout: 5 * time.Second},
	}
	errc := make(chan error, len(servers))
	for _, s := range servers {
		go func() { errc <- s.ListenAndServe() }()
	}
	logger.Printf("1. 先登入 http://%s/login", apiAddr)
	logger.Printf("2. 允許的前端 http://%s/", appAddr)
	logger.Printf("3. 不允許的前端 http://%s/", otherAddr)
	logger.Fatal(<-errc)
}
