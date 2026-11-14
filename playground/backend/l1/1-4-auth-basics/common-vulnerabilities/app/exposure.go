package app

import "net/http"

// meUnsafe encodes the storage model as-is, so every field leaves the server.
func (s *Server) meUnsafe(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.writeJSON(w, s.user)
}

// meView lists, field by field, what a client may see.
type meView struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func (s *Server) meSafe(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.writeJSON(w, meView{ID: s.user.ID, Email: s.user.Email})
}

func (s *Server) lookupUnsafe(w http.ResponseWriter, r *http.Request) {
	if err := lookup(r.URL.Query().Get("slug")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) lookupSafe(w http.ResponseWriter, r *http.Request) {
	if err := lookup(r.URL.Query().Get("slug")); err != nil {
		// Operators get the detail in the log; the client gets nothing to work with.
		s.log.Error("lookup", "err", err)
		http.Error(w, errLookup.Error(), http.StatusInternalServerError)
	}
}
