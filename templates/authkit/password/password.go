// Package password hashes and verifies passwords.
package password

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash = errors.New("password: invalid encoded hash")
	// ErrUnknownPepper means a hash names a pepper this Hasher was not given:
	// a configuration error, not a wrong password.
	ErrUnknownPepper = errors.New("password: hash uses an unknown pepper")
)

// Hasher hashes passwords and verifies them against stored hashes.
// Verify returns (false, nil) on mismatch, ErrInvalidHash on a malformed hash
// and ErrUnknownPepper when the hash's pepper is not configured.
type Hasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
	// NeedsRehash reports whether encoded was made with other parameters or
	// another pepper than Hash uses now. After a successful Verify, store a
	// fresh Hash of the password so old hashes upgrade as users log in.
	NeedsRehash(encoded string) bool
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

// Upper bounds on parameters read from a stored hash, so a corrupt or
// tampered row cannot make Verify allocate gigabytes or spin for minutes.
const (
	maxMemoryKiB = 256 * 1024
	maxTimeCost  = 10
	maxThreads   = 16
)

type argon2id struct {
	pepperID string            // "" = no pepper
	peppers  map[string][]byte // id -> pepper, current one included
}

// NewArgon2id returns a Hasher producing PHC strings:
//
//	$argon2id$v=19$m=...,t=...,p=...[,k=<pepper id>]$salt$hash
//
// pepper is optional (nil = none); keep it outside the database. It signs
// new hashes. previous peppers only verify, so a pepper can be rotated: make
// the new one current, keep the old one in previous until NeedsRehash has
// moved every hash off it, then drop it.
//
// The k parameter names the pepper by a truncated hash of it, never the
// pepper itself. A hash without k was made with no pepper.
func NewArgon2id(pepper []byte, previous ...[]byte) Hasher {
	a := argon2id{peppers: map[string][]byte{}}
	for _, p := range append([][]byte{pepper}, previous...) {
		if len(p) > 0 {
			a.peppers[pepperID(p)] = append([]byte(nil), p...)
		}
	}
	if len(pepper) > 0 {
		a.pepperID = pepperID(pepper)
	}
	return a
}

func pepperID(p []byte) string {
	sum := sha256.Sum256(append([]byte("authkit-pepper:"), p...))
	return hex.EncodeToString(sum[:6])
}

// applyPepper binds the password to the pepper with HMAC-SHA256.
func applyPepper(pepper []byte, password string) []byte {
	if len(pepper) == 0 {
		return []byte(password)
	}
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(password))
	return mac.Sum(nil)
}

func (a argon2id) Hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey(applyPepper(a.peppers[a.pepperID], password), salt, timeCost, memoryKiB, threads, keyLen)
	params := fmt.Sprintf("m=%d,t=%d,p=%d", memoryKiB, timeCost, threads)
	if a.pepperID != "" {
		params += ",k=" + a.pepperID
	}
	return fmt.Sprintf("$argon2id$v=%d$%s$%s$%s",
		argon2.Version, params,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// decoded is a parsed PHC string.
type decoded struct {
	m, t     uint32
	p        uint8
	pepperID string
	salt     []byte
	key      []byte
}

func decode(encoded string) (decoded, error) {
	var d decoded
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return d, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return d, ErrInvalidHash
	}
	params := parts[3]
	if i := strings.Index(params, ",k="); i >= 0 {
		params, d.pepperID = params[:i], params[i+len(",k="):]
		if d.pepperID == "" {
			return d, ErrInvalidHash
		}
	}
	// Sscanf ignores trailing input, so re-format and compare to reject it.
	if _, err := fmt.Sscanf(params, "m=%d,t=%d,p=%d", &d.m, &d.t, &d.p); err != nil ||
		params != fmt.Sprintf("m=%d,t=%d,p=%d", d.m, d.t, d.p) ||
		d.m == 0 || d.m > maxMemoryKiB || d.t == 0 || d.t > maxTimeCost || d.p == 0 || d.p > maxThreads {
		return d, ErrInvalidHash
	}
	var err error
	if d.salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return d, ErrInvalidHash
	}
	if d.key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil || len(d.key) == 0 {
		return d, ErrInvalidHash
	}
	return d, nil
}

func (a argon2id) Verify(password, encoded string) (bool, error) {
	d, err := decode(encoded)
	if err != nil {
		return false, err
	}
	var pepper []byte
	if d.pepperID != "" {
		var ok bool
		if pepper, ok = a.peppers[d.pepperID]; !ok {
			return false, ErrUnknownPepper
		}
	}
	got := argon2.IDKey(applyPepper(pepper, password), d.salt, d.t, d.m, d.p, uint32(len(d.key)))
	return subtle.ConstantTimeCompare(got, d.key) == 1, nil
}

func (a argon2id) NeedsRehash(encoded string) bool {
	d, err := decode(encoded)
	return err != nil || d.m != memoryKiB || d.t != timeCost || d.p != threads ||
		len(d.salt) != saltLen || len(d.key) != keyLen || d.pepperID != a.pepperID
}
