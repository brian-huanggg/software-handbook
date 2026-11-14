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
	if cfg.Addr != ":8080" || cfg.Issuer != "authkit" || cfg.InsecureCookies || cfg.Pepper != nil {
		t.Fatalf("defaults wrong: %+v", cfg)
	}

	over := map[string]string{"ADDR": ":9", "JWT_ISSUER": "me", "COOKIE_INSECURE": "1", "PASSWORD_PEPPER": "pp"}
	for k, v := range good {
		over[k] = v
	}
	cfg, err = loadConfig(getenv(over))
	if err != nil || cfg.Addr != ":9" || cfg.Issuer != "me" || !cfg.InsecureCookies || string(cfg.Pepper) != "pp" {
		t.Fatalf("overrides wrong: %+v, %v", cfg, err)
	}
}

func TestLoadConfig_FailsFast(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"no db":        {"JWT_SECRET": strings.Repeat("k", 32)},
		"no secret":    {"DATABASE_URL": "x"},
		"short secret": {"DATABASE_URL": "x", "JWT_SECRET": "short"},
	} {
		if _, err := loadConfig(getenv(m)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
