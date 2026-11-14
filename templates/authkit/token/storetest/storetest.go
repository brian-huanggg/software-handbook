// Package storetest is a conformance suite shared by every token.RefreshStore.
package storetest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"authkit/token"
)

// Factory returns a fresh store and a function creating a user ID valid for
// that store (the Postgres store has a foreign key to users).
type Factory func(t *testing.T) (s token.RefreshStore, newUser func() string)

func hash() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

func fresh(user, family string) token.Refresh {
	return token.Refresh{
		IDHash: hash(), UserID: user, FamilyID: family,
		ExpiresAt:       time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond),
		FamilyExpiresAt: time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond),
	}
}

func Run(t *testing.T, f Factory) {
	ctx := context.Background()

	t.Run("CreateGet", func(t *testing.T) {
		s, newUser := f(t)
		want := fresh(newUser(), uuid.NewString())
		if err := s.Create(ctx, want); err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(ctx, want.IDHash)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.IDHash, want.IDHash) || got.UserID != want.UserID ||
			got.FamilyID != want.FamilyID || got.UsedAt != nil || !got.ExpiresAt.Equal(want.ExpiresAt) ||
			!got.FamilyExpiresAt.Equal(want.FamilyExpiresAt) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("GetMissing", func(t *testing.T) {
		s, _ := f(t)
		if _, err := s.Get(ctx, hash()); !errors.Is(err, token.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("CreateDuplicate", func(t *testing.T) {
		s, newUser := f(t)
		r := fresh(newUser(), uuid.NewString())
		_ = s.Create(ctx, r)
		if err := s.Create(ctx, r); !errors.Is(err, token.ErrDuplicate) {
			t.Fatalf("got %v, want ErrDuplicate", err)
		}
	})

	t.Run("Rotate", func(t *testing.T) {
		s, newUser := f(t)
		user, fam := newUser(), uuid.NewString()
		old, next := fresh(user, fam), fresh(user, fam)
		_ = s.Create(ctx, old)
		at := time.Now().UTC().Truncate(time.Microsecond)

		ok, err := s.Rotate(ctx, old.IDHash, at, next)
		if err != nil || !ok {
			t.Fatalf("first Rotate = %v, %v; want true", ok, err)
		}
		got, _ := s.Get(ctx, old.IDHash)
		if got.UsedAt == nil || !got.UsedAt.Equal(at) {
			t.Fatalf("old UsedAt = %v, want %v", got.UsedAt, at)
		}
		if got, err := s.Get(ctx, next.IDHash); err != nil || got.UsedAt != nil || got.FamilyID != fam {
			t.Fatalf("next = %+v, %v; want stored and unused", got, err)
		}

		again := fresh(user, fam)
		if ok, err := s.Rotate(ctx, old.IDHash, at, again); err != nil || ok {
			t.Fatalf("second Rotate = %v, %v; want false", ok, err)
		}
		if _, err := s.Get(ctx, again.IDHash); !errors.Is(err, token.ErrNotFound) {
			t.Fatalf("losing Rotate stored its token: %v", err)
		}
		if ok, err := s.Rotate(ctx, hash(), at, fresh(user, fam)); err != nil || ok {
			t.Fatalf("Rotate on missing = %v, %v; want false, nil", ok, err)
		}
	})

	// A failed insert must not consume the old token, so the client can retry.
	t.Run("RotateIsAllOrNothing", func(t *testing.T) {
		s, newUser := f(t)
		user, fam := newUser(), uuid.NewString()
		old, taken := fresh(user, fam), fresh(user, fam)
		_ = s.Create(ctx, old)
		_ = s.Create(ctx, taken)

		next := fresh(user, fam)
		next.IDHash = append([]byte(nil), taken.IDHash...) // collides
		if ok, err := s.Rotate(ctx, old.IDHash, time.Now(), next); ok || !errors.Is(err, token.ErrDuplicate) {
			t.Fatalf("Rotate onto existing hash = %v, %v; want false, ErrDuplicate", ok, err)
		}
		if got, _ := s.Get(ctx, old.IDHash); got.UsedAt != nil {
			t.Fatal("old token was consumed although its replacement was not stored")
		}
	})

	// Exactly one concurrent caller may win the rotation.
	t.Run("RotateIsAtomic", func(t *testing.T) {
		s, newUser := f(t)
		user, fam := newUser(), uuid.NewString()
		old := fresh(user, fam)
		_ = s.Create(ctx, old)

		var wins atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if ok, err := s.Rotate(ctx, old.IDHash, time.Now(), fresh(user, fam)); err == nil && ok {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		if wins.Load() != 1 {
			t.Fatalf("%d winners, want exactly 1", wins.Load())
		}
	})

	t.Run("DeleteExpired", func(t *testing.T) {
		s, newUser := f(t)
		user := newUser()
		live, dead := fresh(user, uuid.NewString()), fresh(user, uuid.NewString())
		dead.ExpiresAt = time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
		for _, r := range []token.Refresh{live, dead} {
			if err := s.Create(ctx, r); err != nil {
				t.Fatal(err)
			}
		}
		n, err := s.DeleteExpired(ctx, time.Now())
		if err != nil || n < 1 {
			t.Fatalf("DeleteExpired = %d, %v; want >= 1", n, err)
		}
		if _, err := s.Get(ctx, dead.IDHash); !errors.Is(err, token.ErrNotFound) {
			t.Fatalf("expired token survived: %v", err)
		}
		if _, err := s.Get(ctx, live.IDHash); err != nil {
			t.Fatalf("live token was deleted: %v", err)
		}
	})

	t.Run("DeleteFamily", func(t *testing.T) {
		s, newUser := f(t)
		user, fam, other := newUser(), uuid.NewString(), uuid.NewString()
		a, b, c := fresh(user, fam), fresh(user, fam), fresh(user, other)
		for _, r := range []token.Refresh{a, b, c} {
			if err := s.Create(ctx, r); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.DeleteFamily(ctx, fam); err != nil {
			t.Fatal(err)
		}
		for _, r := range []token.Refresh{a, b} {
			if _, err := s.Get(ctx, r.IDHash); !errors.Is(err, token.ErrNotFound) {
				t.Fatalf("family member survived: %v", err)
			}
		}
		if _, err := s.Get(ctx, c.IDHash); err != nil {
			t.Fatalf("other family was deleted: %v", err)
		}
		if err := s.DeleteFamily(ctx, uuid.NewString()); err != nil {
			t.Fatalf("deleting unknown family should be a no-op: %v", err)
		}
	})

	t.Run("StoredKeyIsCopy", func(t *testing.T) {
		s, newUser := f(t)
		r := fresh(newUser(), uuid.NewString())
		orig := append([]byte(nil), r.IDHash...)
		_ = s.Create(ctx, r)
		r.IDHash[0] ^= 0xff
		if _, err := s.Get(ctx, orig); err != nil {
			t.Fatalf("stored key must be a copy: %v", err)
		}
	})
}
