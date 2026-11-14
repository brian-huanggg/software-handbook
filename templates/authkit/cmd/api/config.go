package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
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
	Addr        string
	DatabaseURL string
	LogJSON     bool // LOG_FORMAT=json; default is text
	JWTSecret   []byte
	// JWTPreviousSecrets verify tokens signed before a key rotation; they
	// never sign. Comma-separated in JWT_PREVIOUS_SECRETS.
	JWTPreviousSecrets [][]byte
	Issuer             string
	Pepper             []byte // optional password pepper, kept out of the DB
	// PreviousPeppers still verify old hashes, which are re-hashed with
	// Pepper on the next login. Comma-separated in PASSWORD_PREVIOUS_PEPPERS.
	PreviousPeppers [][]byte
	InsecureCookies bool // dev only: drop the Secure flag
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
	for _, p := range strings.Split(getenv("PASSWORD_PREVIOUS_PEPPERS"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			c.PreviousPeppers = append(c.PreviousPeppers, []byte(p))
		}
	}
	switch f := getenv("LOG_FORMAT"); f {
	case "", "text":
	case "json":
		c.LogJSON = true
	default:
		return config{}, fmt.Errorf("LOG_FORMAT must be text or json (got %q)", f)
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
	for _, k := range strings.Split(getenv("JWT_PREVIOUS_SECRETS"), ",") {
		if k = strings.TrimSpace(k); k == "" {
			continue
		}
		if len(k) < 32 {
			return config{}, fmt.Errorf("JWT_PREVIOUS_SECRETS: each secret must be at least 32 bytes (got %d)", len(k))
		}
		c.JWTPreviousSecrets = append(c.JWTPreviousSecrets, []byte(k))
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
