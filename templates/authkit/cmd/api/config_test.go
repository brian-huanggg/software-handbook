package main

import (
	"strings"
	"testing"
)

func getenv(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoadConfig(t *testing.T) {
	good := map[string]string{
		"DATABASE_URL": "postgres://x",
		"JWT_SECRET":   strings.Repeat("k", 32),
	}
	cfg, err := loadConfig(getenv(good))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.Issuer != "authkit" || cfg.InsecureCookies || cfg.Pepper != nil || cfg.JWTPreviousSecrets != nil ||
		cfg.AuthIPBurst != defaultAuthIPBurst || cfg.LoginEmailBurst != defaultLoginEmailBurst {
		t.Fatalf("defaults wrong: %+v", cfg)
	}

	over := map[string]string{"ADDR": ":9", "JWT_ISSUER": "me", "COOKIE_INSECURE": "1", "PASSWORD_PEPPER": "pp",
		"RATE_LIMIT_IP_BURST": "100", "RATE_LIMIT_EMAIL_BURST": "50",
		"JWT_PREVIOUS_SECRETS":      strings.Repeat("a", 32) + ", " + strings.Repeat("b", 32),
		"PASSWORD_PREVIOUS_PEPPERS": "p1,p2"}
	for k, v := range good {
		over[k] = v
	}
	cfg, err = loadConfig(getenv(over))
	if err != nil || cfg.Addr != ":9" || cfg.Issuer != "me" || !cfg.InsecureCookies || string(cfg.Pepper) != "pp" ||
		cfg.AuthIPBurst != 100 || cfg.LoginEmailBurst != 50 ||
		len(cfg.JWTPreviousSecrets) != 2 || string(cfg.JWTPreviousSecrets[1]) != strings.Repeat("b", 32) ||
		len(cfg.PreviousPeppers) != 2 || string(cfg.PreviousPeppers[0]) != "p1" {
		t.Fatalf("overrides wrong: %+v, %v", cfg, err)
	}
}

func TestLoadConfig_CORSOrigins(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "x", "JWT_SECRET": strings.Repeat("k", 32)}
	with := func(v string) map[string]string {
		m := map[string]string{"CORS_ORIGINS": v}
		for k, val := range base {
			m[k] = val
		}
		return m
	}
	cfg, err := loadConfig(getenv(with(" http://localhost:3000 , https://app.example.com ,")))
	if err != nil || len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "http://localhost:3000" || cfg.AllowedOrigins[1] != "https://app.example.com" {
		t.Fatalf("got %+v, %v", cfg.AllowedOrigins, err)
	}
	if cfg, err := loadConfig(getenv(base)); err != nil || len(cfg.AllowedOrigins) != 0 {
		t.Fatalf("unset should mean none: %+v, %v", cfg.AllowedOrigins, err)
	}
	for _, bad := range []string{"*", "http://localhost:3000/", "localhost:3000", "http://a,*"} {
		if _, err := loadConfig(getenv(with(bad))); err == nil {
			t.Errorf("CORS_ORIGINS=%q should be rejected", bad)
		}
	}
}

func TestLoadConfig_FailsFast(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"no db":          {"JWT_SECRET": strings.Repeat("k", 32)},
		"no secret":      {"DATABASE_URL": "x"},
		"short secret":   {"DATABASE_URL": "x", "JWT_SECRET": "short"},
		"short previous": {"DATABASE_URL": "x", "JWT_SECRET": strings.Repeat("k", 32), "JWT_PREVIOUS_SECRETS": "short"},
		"zero burst":     {"DATABASE_URL": "x", "JWT_SECRET": strings.Repeat("k", 32), "RATE_LIMIT_IP_BURST": "0"},
		"bad burst":      {"DATABASE_URL": "x", "JWT_SECRET": strings.Repeat("k", 32), "RATE_LIMIT_EMAIL_BURST": "lots"},
	} {
		if _, err := loadConfig(getenv(m)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
