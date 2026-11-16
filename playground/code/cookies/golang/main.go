package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	id, err := createSession()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    id,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful, cookie has been set.",
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", loginHandler)
	mux.HandleFunc("/", http.NotFound) // anything else: 404

	log.Println("App is listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}
