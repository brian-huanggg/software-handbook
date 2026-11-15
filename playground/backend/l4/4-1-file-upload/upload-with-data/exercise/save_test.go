//go:build exercise

package exercise

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	commitErr error
	calls     []string
}

func (f *fakeStore) Commit(_, key string) error {
	f.calls = append(f.calls, "commit:"+key)
	return f.commitErr
}
func (f *fakeStore) Remove(key string) error { f.calls = append(f.calls, "remove:"+key); return nil }

type fakeRepo struct {
	err   error
	calls int
}

func (f *fakeRepo) Create(context.Context, string) error { f.calls++; return f.err }

func TestSaveOK(t *testing.T) {
	s, r := &fakeStore{}, &fakeRepo{}
	if err := Save(context.Background(), s, r, "/tmp/x", "k1"); err != nil {
		t.Fatal(err)
	}
	if len(s.calls) != 1 || s.calls[0] != "commit:k1" || r.calls != 1 {
		t.Fatalf("store %v, repo %d", s.calls, r.calls)
	}
}

func TestSaveCommitFails(t *testing.T) {
	boom := errors.New("disk full")
	s, r := &fakeStore{commitErr: boom}, &fakeRepo{}
	if err := Save(context.Background(), s, r, "/tmp/x", "k1"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if r.calls != 0 {
		t.Fatal("repo must not be called when the move failed")
	}
}

func TestSaveInsertFailsRemovesFile(t *testing.T) {
	boom := errors.New("db down")
	s, r := &fakeStore{}, &fakeRepo{err: boom}
	if err := Save(context.Background(), s, r, "/tmp/x", "k1"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if len(s.calls) != 2 || s.calls[1] != "remove:k1" {
		t.Fatalf("store calls = %v", s.calls)
	}
}
