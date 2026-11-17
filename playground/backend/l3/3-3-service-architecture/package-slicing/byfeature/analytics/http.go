package analytics

import (
	"encoding/json"
	"net/http"
)

func (r *Recorder) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /stats/top", r.top)
}

func (r *Recorder) top(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(r.Top(10)); err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
	}
}
