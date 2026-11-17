// Command package-slicing serves the feature-style URL shortener on :8080.
// The layered version lives in bylayer/ and is exercised by its tests.
package main

import (
	"log"
	"net/http"

	"handbook/playground/backend/package-slicing/byfeature"
)

func main() {
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", byfeature.NewMux()); err != nil {
		log.Fatal(err)
	}
}
