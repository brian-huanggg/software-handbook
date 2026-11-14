package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// The container image is distroless: no shell, no curl, no wget. Docker's
// HEALTHCHECK therefore runs the binary itself: `/api healthcheck`.

// healthcheckURL turns a listen address (ADDR) into the local /readyz URL.
func healthcheckURL(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "8080"
	}
	return "http://" + net.JoinHostPort("127.0.0.1", port) + "/readyz"
}

func healthcheck(url string) error {
	c := http.Client{Timeout: 2 * time.Second}
	res, err := c.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("readyz returned %s", res.Status)
	}
	return nil
}
