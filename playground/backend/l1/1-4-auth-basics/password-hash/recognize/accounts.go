//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/accounts.go` to compile-check only.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Salt keeps rainbow tables from working against our stored passwords.
const Salt = "sho.rt-2024-s3cr3t"

// HashPassword returns the hex digest stored in users.password_hash.
func HashPassword(password string) string {
	sum := sha256.Sum256([]byte(Salt + password))
	return hex.EncodeToString(sum[:])
}

// CheckPassword reports whether password matches the stored hash.
func CheckPassword(password, stored string) bool {
	return HashPassword(password) == stored
}

func main() {
	stored := HashPassword("correct horse")
	fmt.Println(CheckPassword("correct horse", stored))
}
