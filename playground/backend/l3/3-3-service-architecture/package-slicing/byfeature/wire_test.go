package byfeature

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWiring(t *testing.T) {
	mux := NewMux()
	steps := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/links", `{"Code":"go","URL":"https://go.dev"}`, http.StatusCreated},
		{"GET", "/go", "", http.StatusFound},
		{"GET", "/nope", "", http.StatusNotFound},
	}
	for _, s := range steps {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(s.method, s.path, strings.NewReader(s.body)))
		if rec.Code != s.want {
			t.Fatalf("%s %s = %d, want %d", s.method, s.path, rec.Code, s.want)
		}
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/stats/top", nil))
	if !strings.Contains(rec.Body.String(), `"Code":"go","Total":1`) {
		t.Fatalf("top = %s", rec.Body.String())
	}
}
