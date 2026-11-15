package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// maxHeaderBytes caps request line plus headers. Without a cap, a client can
// keep sending header lines and make the server hold them in memory.
const maxHeaderBytes = 8 << 10

var (
	ErrMalformed      = errors.New("malformed request")
	ErrHeaderTooLarge = errors.New("header too large")
)

// Request is the part of an HTTP/1.1 request this example understands.
type Request struct {
	Method string
	Target string
	Proto  string
	Header map[string]string // keys are lower-case
	Body   []byte
}

// Response is what a Handler returns; the server turns it into bytes.
type Response struct {
	Status int
	Header map[string]string
	Body   string
}

// Handler is the only place that knows about business rules.
type Handler func(*Request) Response

// Server owns the connection lifecycle: accept, read, dispatch, write, reuse.
type Server struct {
	Handler Handler
	// ReadTimeout is the time a client has to deliver one whole request
	// once its first byte has arrived.
	ReadTimeout time.Duration
	// IdleTimeout is how long a kept-alive connection may wait for the next request.
	IdleTimeout time.Duration
	// MaxBodyBytes is reserved for exercise 1; zero means no limit.
	MaxBodyBytes int64
}

func NewServer(h Handler) *Server {
	return &Server{Handler: h, ReadTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
}

// Serve accepts connections until the listener is closed.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	for {
		// Waiting for the next request is idle time, not slow-client time,
		// so the two phases get different deadlines.
		if err := conn.SetReadDeadline(time.Now().Add(s.IdleTimeout)); err != nil {
			return
		}
		if _, err := br.Peek(1); err != nil {
			return
		}
		if err := conn.SetReadDeadline(time.Now().Add(s.ReadTimeout)); err != nil {
			return
		}
		req, err := ReadRequest(br)
		if err != nil {
			if status := statusFor(err); status != 0 {
				// The connection state is unknown after a parse error, so close it;
				// a failed write changes nothing because we are closing anyway.
				_ = writeResponse(conn, Response{Status: status}, false)
			}
			return
		}

		keepAlive := wantsKeepAlive(req)
		if err := writeResponse(conn, s.Handler(req), keepAlive); err != nil || !keepAlive {
			return
		}
	}
}

// ReadRequest parses one request: request line, headers, then the body.
func ReadRequest(br *bufio.Reader) (*Request, error) {
	budget := maxHeaderBytes
	line, err := readLine(br, &budget)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(line, " ")
	if len(parts) != 3 || !strings.HasPrefix(parts[2], "HTTP/1.") {
		return nil, ErrMalformed
	}
	req := &Request{Method: parts[0], Target: parts[1], Proto: parts[2], Header: map[string]string{}}

	for {
		line, err := readLine(br, &budget)
		if err != nil {
			return nil, err
		}
		if line == "" { // the blank line ends the headers
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, ErrMalformed
		}
		req.Header[strings.ToLower(name)] = strings.TrimSpace(value)
	}

	if cl, ok := req.Header["content-length"]; ok {
		n, err := strconv.Atoi(cl)
		if err != nil || n < 0 {
			return nil, ErrMalformed
		}
		req.Body = make([]byte, n)
		if _, err := io.ReadFull(br, req.Body); err != nil {
			return nil, err
		}
	}
	return req, nil
}

// readLine reads one line and charges it to the header budget. A line longer
// than the bufio buffer makes ReadSlice fail instead of growing without bound.
func readLine(br *bufio.Reader, budget *int) (string, error) {
	b, err := br.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return "", ErrHeaderTooLarge
	}
	if err != nil {
		return "", err
	}
	*budget -= len(b)
	if *budget < 0 {
		return "", ErrHeaderTooLarge
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// wantsKeepAlive follows HTTP/1.1 defaults: reuse unless the client says close.
func wantsKeepAlive(req *Request) bool {
	conn := strings.ToLower(req.Header["connection"])
	if req.Proto == "HTTP/1.0" {
		return conn == "keep-alive"
	}
	return conn != "close"
}

// statusFor maps a read error to a response; 0 means just close the connection.
func statusFor(err error) int {
	var ne net.Error
	switch {
	case errors.Is(err, ErrHeaderTooLarge):
		return 431
	case errors.Is(err, ErrMalformed):
		return 400
	case errors.As(err, &ne) && ne.Timeout():
		return 408
	}
	return 0
}

var statusText = map[int]string{
	200: "OK", 302: "Found", 400: "Bad Request", 404: "Not Found",
	405: "Method Not Allowed", 408: "Request Timeout",
	431: "Request Header Fields Too Large",
}

func writeResponse(w io.Writer, resp Response, keepAlive bool) error {
	conn := "close"
	if keepAlive {
		conn = "keep-alive"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP/1.1 %d %s\r\n", resp.Status, statusText[resp.Status])
	// The client needs Content-Length to know where this response ends,
	// otherwise it cannot reuse the connection for the next one.
	fmt.Fprintf(&b, "Content-Length: %d\r\nConnection: %s\r\n", len(resp.Body), conn)
	for k, v := range resp.Header {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	b.WriteString("\r\n")
	b.WriteString(resp.Body)
	_, err := io.WriteString(w, b.String())
	return err
}

// ShortURLHandler is the v1 redirect rule: GET /<code> answers 302.
func ShortURLHandler(links map[string]string) Handler {
	return func(req *Request) Response {
		if req.Method != "GET" {
			return Response{Status: 405, Header: map[string]string{"Allow": "GET"}}
		}
		target, ok := links[strings.TrimPrefix(req.Target, "/")]
		if !ok {
			return Response{Status: 404, Body: "not found\n"}
		}
		return Response{Status: 302, Header: map[string]string{"Location": target}}
	}
}
