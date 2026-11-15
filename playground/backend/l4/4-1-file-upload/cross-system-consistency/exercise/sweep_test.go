//go:build exercise

package exercise

import (
	"errors"
	"testing"
)

type fakeStore struct{ fail map[string]bool }

func (f *fakeStore) Delete(key string) error {
	if f.fail[key] {
		return errors.New("boom")
	}
	return nil
}

type fakeQueue struct{ keys []string }

func (q *fakeQueue) Pending() []string { return append([]string(nil), q.keys...) }
func (q *fakeQueue) Remove(key string) error {
	for i, k := range q.keys {
		if k == key {
			q.keys = append(q.keys[:i], q.keys[i+1:]...)
		}
	}
	return nil
}

func TestSweepKeepsFailedRowAndContinues(t *testing.T) {
	st := &fakeStore{fail: map[string]bool{"a": true}}
	q := &fakeQueue{keys: []string{"a", "b", "c"}}

	if err := Sweep(st, q); err == nil {
		t.Fatal("want the delete error returned")
	}
	if len(q.keys) != 1 || q.keys[0] != "a" {
		t.Fatalf("queue = %v, want [a]", q.keys)
	}

	st.fail = nil
	if err := Sweep(st, q); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if len(q.keys) != 0 {
		t.Fatalf("queue = %v, want empty", q.keys)
	}
}
