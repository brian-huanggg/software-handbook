// Package respond writes JSON responses. Handlers and middleware both use it,
// so every error a client sees has the same shape: {"error": "..."}.
package respond

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as JSON with the given status. Responses are never cached:
// they carry tokens, cookies or per-user data.
func JSON(w http.ResponseWriter, status int, v any) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes {"error": msg}. msg reaches the client, so it must never
// contain internal details.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}
