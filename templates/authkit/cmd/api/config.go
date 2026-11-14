package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"authkit/middleware"
)

const (
	sessionTTL = 24 * time.Hour
	refreshTTL = 7 * 24 * time.Hour
)

type config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       []byte
	Issuer          string
	Pepper          []byte // optional password pepper, kept out of the DB
	InsecureCookies bool   // dev only: drop the Secure flag
	// AllowedOrigins are the browser origins that may call the API with
	// credentials (CORS) and submit state-changing requests (CSRF).
	AllowedOrigins []string
}

// loadConfig reads settings from getenv and fails fast on missing secrets,
// so the server never starts with an unsafe default.
func loadConfig(getenv func(string) string) (config, error) {
	c := config{
		Addr:        getenv("ADDR"),
		DatabaseURL: getenv("DATABASE_URL"),
		JWTSecret:   []byte(getenv("JWT_SECRET")),
		Issuer:      getenv("JWT_ISSUER"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.Issuer == "" {
		c.Issuer = "authkit"
	}
	if p := getenv("PASSWORD_PEPPER"); p != "" {
		c.Pepper = []byte(p)
	}
	switch getenv("COOKIE_INSECURE") {
	case "1", "true":
		c.InsecureCookies = true
	}
	for _, o := range strings.Split(getenv("CORS_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o == "" {
			continue
		}
		if err := middleware.ValidateOrigin(o); err != nil {
			return config{}, fmt.Errorf("CORS_ORIGINS: %w", err)
		}
		c.AllowedOrigins = append(c.AllowedOrigins, o)
	}
	if c.DatabaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return config{}, fmt.Errorf("JWT_SECRET is required and must be at least 32 bytes (got %d)", len(c.JWTSecret))
	}
	return c, nil
}
