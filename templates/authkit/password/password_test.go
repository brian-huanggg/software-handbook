package password_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/brian-huanggg/software-handbook/templates/authkit/password"
)

func newHasher() password.Hasher { return password.NewArgon2id(nil) }

func TestHashVerify_RoundTrip(t *testing.T) {
	h := newHasher()
	enc, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$") {
		t.Fatalf("not PHC argon2id: %q", enc)
	}
	ok, err := h.Verify("correct horse", enc)
	if err != nil || !ok {
		t.Fatalf("Verify = %v, %v; want true, nil", ok, err)
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	h := newHasher()
	enc, _ := h.Hash("right")
	ok, err := h.Verify("wrong", enc)
	if err != nil || ok {
		t.Fatalf("Verify = %v, %v; want false, nil", ok, err)
	}
}

func TestHash_SaltIsRandom(t *testing.T) {
	h := newHasher()
	a, _ := h.Hash("same")
	b, _ := h.Hash("same")
	if a == b {
		t.Fatal("two hashes of same password are identical")
	}
}

func TestVerify_InvalidEncoded(t *testing.T) {
	h := newHasher()
	for _, enc := range []string{
		"",
		"plaintext",
		"$bcrypt$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=1$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$bogus$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$!!!",
	} {
		ok, err := h.Verify("x", enc)
		if ok || !errors.Is(err, password.ErrInvalidHash) {
			t.Errorf("Verify(%q) = %v, %v; want false, ErrInvalidHash", enc, ok, err)
		}
	}
}

func TestVerify_RejectsExcessiveParams(t *testing.T) {
	h := newHasher()
	for _, enc := range []string{
		"$argon2id$v=19$m=4194304,t=2,p=1$c2FsdA$aGFzaA", // 4 GiB memory
		"$argon2id$v=19$m=19456,t=1000,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=255$c2FsdA$aGFzaA",
	} {
		ok, err := h.Verify("x", enc)
		if ok || !errors.Is(err, password.ErrInvalidHash) {
			t.Errorf("Verify(%q) = %v, %v; want false, ErrInvalidHash", enc, ok, err)
		}
	}
}

func TestPepper(t *testing.T) {
	withPepper := password.NewArgon2id([]byte("pepper"))
	enc, _ := withPepper.Hash("pw")

	if ok, _ := withPepper.Verify("pw", enc); !ok {
		t.Fatal("same pepper must verify")
	}
	if ok, _ := newHasher().Verify("pw", enc); ok {
		t.Fatal("hash made with pepper must not verify without it")
	}
	if ok, _ := password.NewArgon2id([]byte("other")).Verify("pw", enc); ok {
		t.Fatal("different pepper must not verify")
	}
}

func TestPepperRotation(t *testing.T) {
	old := password.NewArgon2id([]byte("old pepper"))
	enc, _ := old.Hash("pw")

	rotated := password.NewArgon2id([]byte("new pepper"), []byte("old pepper"))
	if ok, err := rotated.Verify("pw", enc); err != nil || !ok {
		t.Fatalf("previous pepper must still verify: %v, %v", ok, err)
	}
	if !rotated.NeedsRehash(enc) {
		t.Fatal("hash under a previous pepper must need a rehash")
	}
	fresh, _ := rotated.Hash("pw")
	if rotated.NeedsRehash(fresh) {
		t.Fatal("fresh hash must not need a rehash")
	}
	if strings.Contains(fresh, "pepper") {
		t.Fatalf("pepper leaked into hash: %q", fresh)
	}

	dropped := password.NewArgon2id([]byte("new pepper"))
	if _, err := dropped.Verify("pw", enc); !errors.Is(err, password.ErrUnknownPepper) {
		t.Fatalf("dropped pepper: err = %v, want ErrUnknownPepper", err)
	}
}

func TestPepper_AddedLater(t *testing.T) {
	enc, _ := newHasher().Hash("pw") // no pepper, so no k parameter
	if strings.Contains(enc, ",k=") {
		t.Fatalf("unpeppered hash has a pepper id: %q", enc)
	}
	h := password.NewArgon2id([]byte("pepper"))
	if ok, err := h.Verify("pw", enc); err != nil || !ok {
		t.Fatalf("unpeppered hash must verify after a pepper is added: %v, %v", ok, err)
	}
	if !h.NeedsRehash(enc) {
		t.Fatal("unpeppered hash must need a rehash once a pepper is set")
	}
}

func TestNeedsRehash_OldParams(t *testing.T) {
	h := newHasher()
	enc, _ := h.Hash("pw")
	if h.NeedsRehash(enc) {
		t.Fatal("current hash needs rehash")
	}
	weaker := strings.Replace(enc, "t=2", "t=1", 1)
	if !h.NeedsRehash(weaker) {
		t.Fatal("hash with old time cost must need a rehash")
	}
	if !h.NeedsRehash("garbage") {
		t.Fatal("malformed hash must need a rehash")
	}
}

func TestVerify_RejectsTrailingParams(t *testing.T) {
	h := newHasher()
	enc, _ := h.Hash("pw")
	bad := strings.Replace(enc, "p=1", "p=1,x=9", 1)
	if _, err := h.Verify("pw", bad); !errors.Is(err, password.ErrInvalidHash) {
		t.Fatalf("err = %v, want ErrInvalidHash", err)
	}
}
