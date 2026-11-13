package passwordhash

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestArgon2idRoundTrip(t *testing.T) {
	pepper := []byte("pepper-from-env")
	h, err := HashArgon2id("correct horse", pepper)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected format: %s", h)
	}
	ok, err := VerifyArgon2id("correct horse", h, pepper)
	if err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	ok, err = VerifyArgon2id("wrong", h, pepper)
	if err != nil || ok {
		t.Fatalf("wrong password must fail, got ok=%v err=%v", ok, err)
	}
}

func TestSamePasswordDifferentHash(t *testing.T) {
	a, _ := HashArgon2id("same", nil)
	b, _ := HashArgon2id("same", nil)
	if a == b {
		t.Fatal("salt must make every hash different")
	}
}

func TestPepperMismatchFails(t *testing.T) {
	h, err := HashArgon2id("pw", []byte("pepper-a"))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyArgon2id("pw", h, []byte("pepper-b"))
	if err != nil || ok {
		t.Fatalf("db leak without pepper must not verify, got ok=%v err=%v", ok, err)
	}
}

func TestInvalidEncoded(t *testing.T) {
	_, err := VerifyArgon2id("pw", "not-a-hash", nil)
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("want ErrInvalidHash, got %v", err)
	}
}

func TestBcryptRoundTripAndLimit(t *testing.T) {
	h, err := HashBcrypt("correct horse", bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyBcrypt("correct horse", h); err != nil || !ok {
		t.Fatalf("want ok, got ok=%v err=%v", ok, err)
	}
	if ok, err := VerifyBcrypt("wrong", h); err != nil || ok {
		t.Fatalf("wrong password must fail, got ok=%v err=%v", ok, err)
	}
	if _, err := HashBcrypt(strings.Repeat("a", 73), bcrypt.MinCost); !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Fatalf("want ErrPasswordTooLong, got %v", err)
	}
}
