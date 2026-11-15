package main

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startServer(t *testing.T, readTimeout time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	})
	s := NewServer(ShortURLHandler(map[string]string{"abc123": "https://example.com/long"}))
	s.ReadTimeout = readTimeout
	go s.Serve(ln) //nolint:errcheck // Serve returns when the listener closes.
	return ln.Addr().String()
}

func dial(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return conn, bufio.NewReader(conn)
}

func send(t *testing.T, conn net.Conn, raw string) {
	t.Helper()
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
}

func readResp(t *testing.T, br *bufio.Reader) *http.Response {
	t.Helper()
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func TestRedirect(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	send(t, conn, "GET /abc123 HTTP/1.1\r\nHost: sho.rt\r\n\r\n")
	resp := readResp(t, br)
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "https://example.com/long" {
		t.Fatalf("got %d, Location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestNotFoundAndMethod(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	send(t, conn, "GET /nope HTTP/1.1\r\nHost: sho.rt\r\n\r\n")
	if got := readResp(t, br).StatusCode; got != 404 {
		t.Fatalf("GET /nope: got %d, want 404", got)
	}
	send(t, conn, "POST /abc123 HTTP/1.1\r\nHost: sho.rt\r\nContent-Length: 2\r\n\r\nhi")
	if got := readResp(t, br).StatusCode; got != 405 {
		t.Fatalf("POST: got %d, want 405", got)
	}
}

func TestKeepAliveReusesConnection(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	for i := 0; i < 3; i++ {
		send(t, conn, "GET /abc123 HTTP/1.1\r\nHost: sho.rt\r\n\r\n")
		if got := readResp(t, br).StatusCode; got != 302 {
			t.Fatalf("request %d on same connection: got %d", i, got)
		}
	}
}

func TestConnectionCloseClosesAfterResponse(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	send(t, conn, "GET /abc123 HTTP/1.1\r\nHost: sho.rt\r\nConnection: close\r\n\r\n")
	readResp(t, br)
	if _, err := br.ReadByte(); err == nil {
		t.Fatal("connection still open after Connection: close")
	}
}

func TestSlowClientGets408(t *testing.T) {
	conn, br := dial(t, startServer(t, 100*time.Millisecond))
	// Send the start of a request, then stall: the Slowloris pattern.
	send(t, conn, "GET /abc123 HTTP/1.1\r\nHost:")
	if got := readResp(t, br).StatusCode; got != 408 {
		t.Fatalf("got %d, want 408", got)
	}
}

func TestHugeHeaderGets431(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	send(t, conn, "GET /abc123 HTTP/1.1\r\nX-Big: "+strings.Repeat("a", 6000)+"\r\n\r\n")
	if got := readResp(t, br).StatusCode; got != 431 {
		t.Fatalf("got %d, want 431", got)
	}
}

func TestMalformedRequestLineGets400(t *testing.T) {
	conn, br := dial(t, startServer(t, time.Second))
	send(t, conn, "HELLO\r\n\r\n")
	if got := readResp(t, br).StatusCode; got != 400 {
		t.Fatalf("got %d, want 400", got)
	}
}

func TestReadRequestParsesBody(t *testing.T) {
	raw := "POST /x HTTP/1.1\r\nHost: a\r\nContent-Length: 5\r\n\r\nhello"
	req, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" || req.Target != "/x" || string(req.Body) != "hello" || req.Header["host"] != "a" {
		t.Fatalf("unexpected request: %+v", req)
	}
}
