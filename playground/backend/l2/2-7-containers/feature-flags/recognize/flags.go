//go:build ignore

// AI 產生：短網址服務 v2 的功能旗標，用來把新的查詢路徑放量給 20% 使用者。
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Flag struct {
	Percent int `json:"percent"`
}

var flags = map[string]Flag{}

func reload(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read flags: %v", err)
		flags = map[string]Flag{"new-lookup-path": {Percent: 100}}
		return
	}
	var next map[string]Flag
	if err := json.Unmarshal(data, &next); err != nil {
		log.Printf("parse flags: %v", err)
		return
	}
	flags = next
}

func enabled(name string) bool {
	f, ok := flags[name]
	if !ok {
		return false
	}
	return rand.Intn(100) < f.Percent
}

func main() {
	reload("flags.json")
	go func() {
		for range time.Tick(time.Second) {
			reload("flags.json")
		}
	}()

	http.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		if enabled("new-lookup-path") {
			w.Write([]byte("new path\n"))
			return
		}
		w.Write([]byte("old path\n"))
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
