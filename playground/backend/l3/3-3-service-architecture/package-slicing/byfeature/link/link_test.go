package link

import (
	"errors"
	"testing"
)

type fakeRecorder struct{ codes []string }

func (f *fakeRecorder) Record(code string) { f.codes = append(f.codes, code) }

func TestResolveRecordsClick(t *testing.T) {
	rec := &fakeRecorder{}
	s := New(rec)
	s.Create("go", "https://go.dev")

	url, err := s.Resolve("go")
	if err != nil || url != "https://go.dev" {
		t.Fatalf("Resolve(go) = %q, %v", url, err)
	}
	if _, err := s.Resolve("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Resolve(nope) err = %v, want ErrNotFound", err)
	}
	if len(rec.codes) != 1 || rec.codes[0] != "go" {
		t.Fatalf("recorded %v, want [go]", rec.codes)
	}
}
