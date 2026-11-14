package main

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"log"
	"time"

	"handbook/playground/backend/jwt-vs-paseto/token"
)

func main() {
	// A fixed seed keeps the output reproducible; real keys come from a KMS.
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	now := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	c := token.Claims{Issuer: "sho.rt", Subject: "42", ExpiresAt: now.Add(15 * time.Minute)}

	jwt, err := token.SignJWT(priv, c)
	if err != nil {
		log.Fatal(err)
	}
	pas, err := token.SignPublic(priv, c, []byte(`{"kid":"2026-11"}`), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("JWT    (EdDSA)     %3d bytes  %.40s...\n", len(jwt), jwt)
	fmt.Printf("PASETO (v4.public) %3d bytes  %.40s...\n", len(pas), pas)

	peeked, err := token.PeekPublic(pas)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("payload readable without any key:", peeked)
}
