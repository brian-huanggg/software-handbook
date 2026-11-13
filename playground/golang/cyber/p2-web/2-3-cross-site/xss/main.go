// Command xss runs a deliberately XSS-prone short-URL service (victim.test)
// next to an attacker's page (evil.test), both on 127.0.0.1. Map the two
// hostnames to 127.0.0.1 in /etc/hosts, pick a -defense, and see which
// payloads still run in alice's browser.
//
// Everything here targets only your own machine. Do not point it at anything else.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

const (
	victimAddr = "127.0.0.1:8080"
	attackAddr = "127.0.0.1:9666"
)

func main() {
	def := flag.String("defense", "none", "server defense: none, encode, csp")
	httpOnly := flag.Bool("httponly", false, "mark the session cookie HttpOnly")
	victimURL := flag.String("victim", "http://victim.test:8080", "victim origin as the browser sees it")
	flag.Parse()

	logger := log.Default()
	vh, err := newVictim(Config{Defense: *def, HttpOnly: *httpOnly}, logger)
	if err != nil {
		log.Fatal(err)
	}

	servers := []*http.Server{
		{Addr: victimAddr, Handler: vh, ReadHeaderTimeout: 5 * time.Second},
		{Addr: attackAddr, Handler: attackPage(*victimURL, logger), ReadHeaderTimeout: 5 * time.Second},
	}
	errc := make(chan error, len(servers))
	for _, s := range servers {
		go func() { errc <- s.ListenAndServe() }()
	}
	logger.Printf("defense=%s httponly=%t", *def, *httpOnly)
	logger.Printf("1. 先登入：%s/login", *victimURL)
	logger.Printf("2. 反射型與 DOM 型：打開 http://evil.test:9666/")
	logger.Printf("3. 儲存型：用 curl POST %s/reports 留下惡意備註，再打開 %s/admin", *victimURL, *victimURL)
	log.Fatal(<-errc)
}
