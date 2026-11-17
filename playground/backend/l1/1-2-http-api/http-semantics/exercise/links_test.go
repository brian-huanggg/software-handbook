//go:build exercise

package exercise

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestPutNewCodeReturns201WithLocation(t *testing.T) {
	h := NewHandler(NewStore())
	rec := do(h, "PUT", "/links/abc", "https://example.com/a")
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/links/abc" {
		t.Fatalf("Location = %q, want /links/abc", got)
	}
}

func TestPutExistingCodeReturns204(t *testing.T) {
	s := NewStore()
	h := NewHandler(s)
	do(h, "PUT", "/links/abc", "https://example.com/a")
	if rec := do(h, "PUT", "/links/abc", "https://example.com/b"); rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, want 204", rec.Code)
	}
	if got := s.links["abc"]; got != "https://example.com/b" {
		t.Fatalf("target = %q, want the replaced value", got)
	}
}

func TestPutEmptyBodyReturns400AndStoresNothing(t *testing.T) {
	s := NewStore()
	h := NewHandler(s)
	if rec := do(h, "PUT", "/links/abc", "  "); rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
	if _, ok := s.links["abc"]; ok {
		t.Fatal("a rejected PUT must not store anything")
	}
}

func TestDeleteTwice(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/abc", "https://example.com/a")
	if rec := do(h, "DELETE", "/links/abc", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("first DELETE: got %d, want 204", rec.Code)
	}
	if rec := do(h, "DELETE", "/links/abc", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE: got %d, want 404", rec.Code)
	}
}
