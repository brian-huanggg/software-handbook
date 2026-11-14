// Package storetest is a conformance suite shared by every session.Store.
package storetest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"authkit/session"
)

// Factory returns a fresh store and a function creating a user ID that is
// valid for that store (the Postgres store has a foreign key to users).
type Factory func(t *testing.T) (s session.Store, newUser func() string)

func hash() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

func Run(t *testing.T, f Factory) {
	ctx := context.Background()

	t.Run("CreateGet", func(t *testing.T) {
		s, newUser := f(t)
		want := session.Session{IDHash: hash(), UserID: newUser(), ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)}
		if err := s.Create(ctx, want); err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(ctx, want.IDHash)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.IDHash, want.IDHash) || got.UserID != want.UserID || !got.ExpiresAt.Equal(want.ExpiresAt) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("GetMissing", func(t *testing.T) {
		s, _ := f(t)
		if _, err := s.Get(ctx, hash()); !errors.Is(err, session.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("CreateDuplicate", func(t *testing.T) {
		s, newUser := f(t)
		sess := session.Session{IDHash: hash(), UserID: newUser(), ExpiresAt: time.Now().Add(time.Hour)}
		if err := s.Create(ctx, sess); err != nil {
			t.Fatal(err)
		}
		if err := s.Create(ctx, sess); !errors.Is(err, session.ErrDuplicate) {
			t.Fatalf("got %v, want ErrDuplicate", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		s, newUser := f(t)
		sess := session.Session{IDHash: hash(), UserID: newUser(), ExpiresAt: time.Now().Add(time.Hour)}
		_ = s.Create(ctx, sess)
		if err := s.Delete(ctx, sess.IDHash); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get(ctx, sess.IDHash); !errors.Is(err, session.ErrNotFound) {
			t.Fatalf("after delete got %v, want ErrNotFound", err)
		}
		if err := s.Delete(ctx, sess.IDHash); err != nil {
			t.Fatalf("second delete should be a no-op, got %v", err)
		}
	})

	t.Run("ReturnsExpiredRows", func(t *testing.T) {
		s, newUser := f(t)
		sess := session.Session{IDHash: hash(), UserID: newUser(), ExpiresAt: time.Now().Add(-time.Hour)}
		_ = s.Create(ctx, sess)
		if _, err := s.Get(ctx, sess.IDHash); err != nil {
			t.Fatalf("store must leave expiry to Manager, got %v", err)
		}
	})

	t.Run("DeleteExpired", func(t *testing.T) {
		s, newUser := f(t)
		user := newUser()
		live := session.Session{IDHash: hash(), UserID: user, ExpiresAt: time.Now().Add(time.Hour)}
		dead := session.Session{IDHash: hash(), UserID: user, ExpiresAt: time.Now().Add(-time.Hour)}
		_ = s.Create(ctx, live)
		_ = s.Create(ctx, dead)
		n, err := s.DeleteExpired(ctx, time.Now())
		if err != nil || n < 1 {
			t.Fatalf("DeleteExpired = %d, %v; want >= 1", n, err)
		}
		if _, err := s.Get(ctx, dead.IDHash); !errors.Is(err, session.ErrNotFound) {
			t.Fatalf("expired session survived: %v", err)
		}
		if _, err := s.Get(ctx, live.IDHash); err != nil {
			t.Fatalf("live session was deleted: %v", err)
		}
	})

	t.Run("GetDoesNotAliasInput", func(t *testing.T) {
		s, newUser := f(t)
		sess := session.Session{IDHash: hash(), UserID: newUser(), ExpiresAt: time.Now().Add(time.Hour)}
		orig := append([]byte(nil), sess.IDHash...)
		_ = s.Create(ctx, sess)
		sess.IDHash[0] ^= 0xff // caller mutates its slice after Create
		got, err := s.Get(ctx, orig)
		if err != nil {
			t.Fatalf("stored key must be a copy: %v", err)
		}
		if !bytes.Equal(got.IDHash, orig) {
			t.Fatal("stored hash changed")
		}
	})
}
