package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"authkit/middleware"
)

const (
	sessionTTL = 24 * time.Hour
	refreshTTL = 7 * 24 * time.Hour

	// janitorInterval is how often expired sessions and refresh tokens are purged.
	janitorInterval = 10 * time.Minute

	// Brute-force limits (token buckets). Per client IP on /register and the
	// logins: a burst of 20, then one every 3s (20/min). Per email on the
	// logins: a burst of 5, then one every 30s (2/min). The bursts can be
	// raised with RATE_LIMIT_IP_BURST / RATE_LIMIT_EMAIL_BURST.
	authIPEvery            = 3 * time.Second
	defaultAuthIPBurst     = 20
	loginEmailEvery        = 30 * time.Second
	defaultLoginEmailBurst = 5
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
	// Burst sizes of the brute-force limiters; always > 0.
	AuthIPBurst     int
	LoginEmailBurst int
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
	var err error
	if c.AuthIPBurst, err = positiveInt(getenv, "RATE_LIMIT_IP_BURST", defaultAuthIPBurst); err != nil {
		return config{}, err
	}
	if c.LoginEmailBurst, err = positiveInt(getenv, "RATE_LIMIT_EMAIL_BURST", defaultLoginEmailBurst); err != nil {
		return config{}, err
	}
	if c.DatabaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return config{}, fmt.Errorf("JWT_SECRET is required and must be at least 32 bytes (got %d)", len(c.JWTSecret))
	}
	return c, nil
}

// positiveInt reads key as an integer > 0, or def when unset. There is no
// "off" value: a limit can be raised, never silently disabled.
func positiveInt(getenv func(string) string, key string, def int) (int, error) {
	v := getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, v)
	}
	return n, nil
}
