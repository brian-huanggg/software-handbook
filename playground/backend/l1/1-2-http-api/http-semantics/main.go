package main

import (
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	srv := NewServer(":8080", NewHandler(NewStore()))
	logger.Println("listening on :8080")
	logger.Fatal(srv.ListenAndServe())
}
