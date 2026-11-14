// Command common-vulnerabilities serves the demo API on localhost.
// Run with -vulnerable to see the unfixed version.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"handbook/playground/backend/common-vulnerabilities/app"
)

func main() {
	vulnerable := flag.Bool("vulnerable", false, "serve the version without fixes")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	s := app.NewServer(logger)
	h := s.Safe()
	if *vulnerable {
		h = s.Vulnerable()
	}

	// Bound to loopback on purpose: the vulnerable version must never be reachable from outside.
	srv := &http.Server{Addr: "127.0.0.1:8080", Handler: h, ReadHeaderTimeout: 5 * time.Second}
	logger.Info("listening", "addr", srv.Addr, "vulnerable", *vulnerable)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("serve", "err", err)
		os.Exit(1)
	}
}
