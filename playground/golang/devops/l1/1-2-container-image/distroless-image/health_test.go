package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeHealthy(t *testing.T) {
	srv := httptest.NewServer(newHandler())
	defer srv.Close()
	if err := probe(srv.URL + "/healthz"); err != nil {
		t.Fatalf("probe: %v", err)
	}
}

func TestProbeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL + "/healthz"
	srv.Close()
	if err := probe(url); err == nil {
		t.Fatal("probe of a closed server should fail")
	}
}
