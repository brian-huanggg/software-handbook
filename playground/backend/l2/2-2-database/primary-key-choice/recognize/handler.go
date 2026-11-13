//go:build ignore

// AI 產生：短網址服務 v2 的 links 資料存取與 API。
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type Link struct {
	ID      string `json:"id"`
	OwnerID int64  `json:"owner_id"`
	Code    string `json:"code"`
	URL     string `json:"url"`
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func createLink(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in Link
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		in.ID = newID()
		_, err := db.Exec(
			"INSERT INTO links (id, owner_id, code, url) VALUES (?, ?, ?, ?)",
			in.ID, in.OwnerID, in.Code, in.URL,
		)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(in)
	}
}

func getLink(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var l Link
		err := db.QueryRow(
			"SELECT id, owner_id, code, url FROM links WHERE id = ?", r.PathValue("id"),
		).Scan(&l.ID, &l.OwnerID, &l.Code, &l.URL)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(l)
	}
}

func main() {}
