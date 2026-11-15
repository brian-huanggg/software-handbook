package filecheck

import (
	"io"
	"net/http"
)

// ServeStored writes a stored upload with the type the server decided at
// upload time, and forbids the browser from guessing a different one.
func ServeStored(w http.ResponseWriter, info Info, body io.Reader) error {
	w.Header().Set("Content-Type", info.Type)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, err := io.Copy(w, body)
	return err
}
