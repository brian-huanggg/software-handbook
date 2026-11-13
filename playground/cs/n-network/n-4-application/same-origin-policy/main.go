// Command same-origin-policy runs two servers on two ports of the same
// machine. Different ports mean different origins, which is enough to watch
// the browser's same-origin policy without registering any domain.
package main

import (
	"log"
	"net/http"
	"time"
)

const (
	apiAddr  = "127.0.0.1:8080"
	pageAddr = "127.0.0.1:9090"
)

func main() {
	logger := log.Default()
	servers := []*http.Server{
		{Addr: apiAddr, Handler: newAPI(newLinkStore(), logger), ReadHeaderTimeout: 5 * time.Second},
		{Addr: pageAddr, Handler: newPage("http://"+apiAddr, logger), ReadHeaderTimeout: 5 * time.Second},
	}
	errc := make(chan error, len(servers))
	for _, s := range servers {
		go func() { errc <- s.ListenAndServe() }()
	}
	logger.Printf("API  http://%s/api/links", apiAddr)
	logger.Printf("頁面 http://%s/ （用瀏覽器開這個）", pageAddr)
	logger.Fatal(<-errc)
}
