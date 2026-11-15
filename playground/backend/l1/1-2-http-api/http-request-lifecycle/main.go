// Command http-request-lifecycle serves the short-URL redirect over a
// hand-written HTTP/1.1 loop on a raw TCP listener.
package main

import (
	"log"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	links := map[string]string{"abc123": "https://example.com/some/long/path"}
	log.Printf("listening on %s", ln.Addr())
	log.Fatal(NewServer(ShortURLHandler(links)).Serve(ln))
}
