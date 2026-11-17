package main

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const sessionTTL = 3600 * time.Second

type session struct {
	status    string
	createdAt time.Time
	expiresIn time.Duration
}

var (
	mu       sync.Mutex
	sessions = map[string]session{}
)

func createSession() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(b) // convert bytes to string

	mu.Lock()
	defer mu.Unlock()

	sessions[id] = session{
		status:    "authenticated",
		createdAt: time.Now(),
		expiresIn: sessionTTL,
	}

	return id, nil
}
