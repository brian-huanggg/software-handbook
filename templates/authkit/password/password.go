// Package password hashes and verifies passwords.
package password

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidHash = errors.New("password: invalid encoded hash")

// Hasher hashes passwords and verifies them against stored hashes.
// Verify returns (false, nil) on mismatch and ErrInvalidHash on a malformed hash.
type Hasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
}

// Argon2id parameters follow the OWASP Password Storage Cheat Sheet minimum.
// They are written into the output, so raising them later still verifies old hashes.
const (
	memoryKiB = 19 * 1024
	timeCost  = 2
	threads   = 1
	saltLen   = 16
	keyLen    = 32
)

type argon2id struct{ pepper []byte }

// NewArgon2id returns a Hasher producing PHC strings:
// $argon2id$v=19$m=...,t=...,p=...$salt$hash
// pepper is optional (nil = none); keep it outside the database.
func NewArgon2id(pepper []byte) Hasher { return argon2id{pepper: pepper} }

// applyPepper binds the password to the pepper with HMAC-SHA256.
func (a argon2id) applyPepper(password string) []byte {
	if len(a.pepper) == 0 {
		return []byte(password)
	}
	mac := hmac.New(sha256.New, a.pepper)
	mac.Write([]byte(password))
	return mac.Sum(nil)
}

func (a argon2id) Hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey(a.applyPepper(password), salt, timeCost, memoryKiB, threads, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, timeCost, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func (a argon2id) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrInvalidHash
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || t == 0 || p == 0 {
		return false, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrInvalidHash
	}
	got := argon2.IDKey(a.applyPepper(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
