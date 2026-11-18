//go:build exercise

package exercise

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seeded() http.Handler {
	return NewHandler(NewStore(
		Link{"a", "https://example.com/1"},
		Link{"b", "https://example.org/2"},
		Link{"c", "https://example.com/3"},
		Link{"d", "https://sub.example.com/4"},
	))
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", target, nil))
	return rec
}

func codes(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var out struct {
		Items []Link `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var b strings.Builder
	for _, l := range out.Items {
		b.WriteString(l.Code)
	}
	return b.String()
}

func TestDefaultsReturnEverything(t *testing.T) {
	if got := codes(t, get(seeded(), "/links")); got != "abcd" {
		t.Fatalf("got %q, want abcd", got)
	}
}

func TestLimitAndOffset(t *testing.T) {
	if got := codes(t, get(seeded(), "/links?limit=2&offset=1")); got != "bc" {
		t.Fatalf("got %q, want bc", got)
	}
}

func TestHostMustMatchExactly(t *testing.T) {
	if got := codes(t, get(seeded(), "/links?host=example.com")); got != "ac" {
		t.Fatalf("got %q, want ac (sub.example.com is a different host)", got)
	}
}

func TestEmptyResultIsEmptyArray(t *testing.T) {
	rec := get(seeded(), "/links?host=nowhere.test")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("got %d %s, want 200 with an empty array", rec.Code, rec.Body)
	}
}

func TestBadParametersAre400(t *testing.T) {
	for _, q := range []string{"limit=0", "limit=101", "limit=x", "offset=-1", "offset=x"} {
		if got := get(seeded(), "/links?"+q).Code; got != http.StatusBadRequest {
			t.Errorf("?%s = %d, want 400", q, got)
		}
	}
}
