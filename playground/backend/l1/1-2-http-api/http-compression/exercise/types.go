//go:build exercise

// Package exercise: finish TypeAllowed so that
// `go test -tags exercise ./exercise/` passes.
package exercise

// TypeAllowed reports whether a Content-Type header value is in the
// allowlist.
//
// Rules:
//   - Ignore parameters: "text/html; charset=utf-8" is "text/html".
//   - Media types are case-insensitive: "Application/JSON" is "application/json".
//   - An entry ending in "/*" matches the whole family: "text/*" matches "text/css".
//   - An empty or malformed value is never allowed.
func TypeAllowed(contentType string, allow []string) bool {
	// TODO
	return false
}
