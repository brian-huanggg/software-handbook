package main

import (
	"fmt"
	"net/http"
	"time"
)

// probe is the container's own health check. A distroless image has no curl or
// wget, so the binary has to be able to check itself.
func probe(url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}
	return nil
}
