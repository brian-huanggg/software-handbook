//go:build ignore

// AI-generated. Not part of the example's tests: run `go vet recognize/verify.go` to compile-check only.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var secret = []byte("change-me")

func parseUserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("malformed token")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(hb, &h); err != nil {
		return "", err
	}

	if h.Alg == "HS256" {
		m := hmac.New(sha256.New, secret)
		m.Write([]byte(parts[0] + "." + parts[1]))
		want := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
		if want != parts[2] {
			return "", errors.New("bad signature")
		}
	}

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var c struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(pb, &c); err != nil {
		return "", err
	}
	return c.Sub, nil
}
