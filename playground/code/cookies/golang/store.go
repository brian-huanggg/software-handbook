package main

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const sessionTTL = 3600 * time.Second // Default: millisecond (ms)

type session struct {
	status    string
	createdAt time.Time
	expiredAt time.Time
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
	id := base64.RawURLEncoding.EncodeToString(b) // Need to convert byte (buffer) to strong

	now := time.Now()
	mu.Lock()
	defer mu.Unlock()

	sessions[id] = session{
		status:    "unauthenticated",
		createdAt: now,
		expiredAt: now.Add(sessionTTL),
	}

	return id, nil
}
