package main

import (
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	token := os.Getenv("API_TOKEN")
	if token == "" {
		logger.Fatal("API_TOKEN is required")
	}
	srv := NewServer(":8080", NewHandler(NewStore(), token, logger))
	logger.Println("listening on :8080")
	logger.Fatal(srv.ListenAndServe())
}
