package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on -addr and exit 0 or 1")
	flag.Parse()

	if *healthcheck {
		if err := probe("http://127.0.0.1" + *addr + "/healthz"); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}

	// Logs go to stderr: the container runtime collects them, no log file needed.
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newHandler()))
}
