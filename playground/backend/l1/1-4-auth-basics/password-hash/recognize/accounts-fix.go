//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/accounts.go` to compile-check only.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// GenerateSalt returns a random salt for password hashing.
func GenerateSalt() string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "Salt cannot be generated: " + err.Error()
	}
	return string(salt)
}

// HashPassword returns the hex digest stored in users.password_hash.
func HashPassword(password string) string {
	key := argon2.IDKey([]byte(password), []byte(GenerateSalt()), 1, 64*1024, 4, 32)
	return string(key)
}

// CheckPassword reports whether password matches the stored hash.
func CheckPassword(password, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(password), []byte(stored)) == 1
}

func main() {
	stored := HashPassword("correct horse")
	fmt.Println(CheckPassword("correct horse", stored))
}
