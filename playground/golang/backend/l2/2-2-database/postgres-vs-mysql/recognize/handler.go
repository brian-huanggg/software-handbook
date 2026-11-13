//go:build ignore

// AI 產生：短網址服務 v2 的「自訂短碼」API，資料庫是 PostgreSQL（預設隔離級別）。
package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type createReq struct {
	OwnerID int64  `json:"owner_id"`
	Code    string `json:"code"`
	URL     string `json:"url"`
}

func createCustomCode(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var n int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM links WHERE code = $1`, req.Code).Scan(&n); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Error(w, "code already taken", http.StatusConflict)
		return
	}

	// 向外部的安全瀏覽服務確認網址沒有被列入黑名單。
	if !isSafeURL(req.URL) {
		http.Error(w, "unsafe url", http.StatusUnprocessableEntity)
		return
	}

	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO links (owner_id, code, url) VALUES ($1, $2, $3)`,
		req.OwnerID, req.Code, req.URL); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// isSafeURL 對外部服務發 HTTP 請求，通常要 100 到 800 毫秒。
func isSafeURL(u string) bool {
	resp, err := http.Get("https://safebrowsing.internal.example/check?url=" + u)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func main() {
	http.HandleFunc("POST /api/links", createCustomCode)
	http.ListenAndServe(":8080", nil)
}
