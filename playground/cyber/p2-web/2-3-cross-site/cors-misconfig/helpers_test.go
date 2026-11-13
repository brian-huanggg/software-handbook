package main

import (
	"io"
	"log"
	"net/http"
)

func discardLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func readBody(res *http.Response) string {
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}
	res.Body.Close()
	return string(b)
}
