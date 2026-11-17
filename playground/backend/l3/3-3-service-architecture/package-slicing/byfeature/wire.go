// Package byfeature wires the feature packages together. It is the only
// place that knows both link and analytics exist.
package byfeature

import (
	"net/http"

	"handbook/playground/backend/package-slicing/byfeature/analytics"
	"handbook/playground/backend/package-slicing/byfeature/link"
)

func NewMux() *http.ServeMux {
	clicks := analytics.New()
	links := link.New(clicks) // *analytics.Recorder satisfies link.ClickRecorder

	mux := http.NewServeMux()
	clicks.Routes(mux)
	links.Routes(mux)
	return mux
}
