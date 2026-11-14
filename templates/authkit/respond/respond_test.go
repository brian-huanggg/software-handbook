package respond_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/brian-huanggg/software-handbook/templates/authkit/respond"
)

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	respond.Error(w, 403, "forbidden")
	if w.Code != 403 {
		t.Fatalf("code %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control %q", cc)
	}
	var body struct{ Error string }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Error != "forbidden" {
		t.Fatalf("body %q err %v", w.Body, err)
	}
}
