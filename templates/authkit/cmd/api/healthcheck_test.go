package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckURL(t *testing.T) {
	for addr, want := range map[string]string{
		"":             "http://127.0.0.1:8080/readyz",
		":8080":        "http://127.0.0.1:8080/readyz",
		":9000":        "http://127.0.0.1:9000/readyz",
		"0.0.0.0:9000": "http://127.0.0.1:9000/readyz",
		"localhost:81": "http://127.0.0.1:81/readyz",
	} {
		if got := healthcheckURL(addr); got != want {
			t.Errorf("healthcheckURL(%q) = %q, want %q", addr, got, want)
		}
	}
}

func TestHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer ok.Close()
	if err := healthcheck(ok.URL); err != nil {
		t.Fatalf("healthy server: %v", err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer bad.Close()
	if healthcheck(bad.URL) == nil {
		t.Fatal("503 must be reported as unhealthy")
	}

	ok.Close()
	if healthcheck(ok.URL) == nil {
		t.Fatal("unreachable server must be reported as unhealthy")
	}
}
