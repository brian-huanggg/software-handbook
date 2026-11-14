package main

import (
	"io"
	"log/slog"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
)

// newLogger builds the process logger: JSON for production log pipelines,
// text for a terminal. Either way every record that carries a request
// context also carries its request_id.
func newLogger(w io.Writer, json bool) *slog.Logger {
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(w, nil)
	} else {
		h = slog.NewTextHandler(w, nil)
	}
	return slog.New(middleware.WithRequestID(h))
}
