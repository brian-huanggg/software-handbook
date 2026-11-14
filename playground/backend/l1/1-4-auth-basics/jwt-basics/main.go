// Command jwt-basics signs a token and shows what is inside it.
package main

import (
	"fmt"
	"log"
	"time"

	"handbook/playground/backend/jwt-basics/jwt"
)

func main() {
	key := []byte("demo-only-secret-demo-only-secret")
	now := time.Now()

	tok, err := jwt.Sign(jwt.Claims{
		Issuer:    "sho.rt",
		Subject:   "42",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(15 * time.Minute).Unix(),
	}, key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("token:  ", tok)

	// Anyone holding the token can read the payload; no key needed.
	payload, err := jwt.Peek(tok)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("payload:", payload)

	c, err := jwt.Verify(tok, key, "sho.rt", now)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("verified sub:", c.Subject)
}
