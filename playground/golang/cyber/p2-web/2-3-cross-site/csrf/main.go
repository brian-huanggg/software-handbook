// Command csrf runs a deliberately CSRF-prone short-URL service (victim.test)
// next to an attacker's page (evil.test), both on 127.0.0.1. Map the two
// hostnames to 127.0.0.1 in /etc/hosts, pick a -defense and a -samesite, and
// see which requests still change where alice's /a1 points.
//
// Everything here targets only your own machine. Do not point it at anything else.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	victimAddr = "127.0.0.1:8080"
	attackAddr = "127.0.0.1:9666"
)

func sameSiteFor(mode string) (http.SameSite, error) {
	switch mode {
	case "default":
		return 0, nil // no attribute: the browser decides
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	}
	return 0, fmt.Errorf("unknown samesite %q (default, lax, strict)", mode)
}

func main() {
	def := flag.String("defense", "none", "server defense: none, token, crosssite")
	ss := flag.String("samesite", "default", "session cookie SameSite: default, lax, strict")
	victimURL := flag.String("victim", "http://victim.test:8080", "victim origin as the browser sees it")
	flag.Parse()

	sameSite, err := sameSiteFor(*ss)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.Default()
	vh, err := newVictim(Config{SameSite: sameSite, Defense: *def}, logger)
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
	logger.Printf("defense=%s samesite=%s", *def, *ss)
	logger.Printf("1. 先登入：%s/login", *victimURL)
	logger.Printf("2. 再打開攻擊頁：http://evil.test:9666/")
	logger.Printf("3. 回 %s/a1 看它指到哪", *victimURL)
	log.Fatal(<-errc)
}
