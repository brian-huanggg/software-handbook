package main

import (
	"encoding/json"
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

func items(t *testing.T, rec *httptest.ResponseRecorder) []Link {
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
	return out.Items
}

func TestRoutes(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/abc", `{"target":"https://example.com/a"}`)

	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"item", "GET", "/links/abc", "", 200},
		{"missing item", "GET", "/links/nope", "", 404},
		{"collection", "GET", "/links", "", 200},
		{"wrong method on item", "POST", "/links/abc", "", 405},
		{"wrong method on collection", "DELETE", "/links", "", 405},
		{"trailing slash is a different URL", "GET", "/links/", "", 404},
		{"case matters", "GET", "/Links", "", 404},
		{"disable missing", "POST", "/links/nope/disable", "", 404},
		{"delete then again", "DELETE", "/links/abc", "", 204},
		{"second delete", "DELETE", "/links/abc", "", 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := do(h, tc.method, tc.path, tc.body).Code; got != tc.want {
				t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, got, tc.want)
			}
		})
	}
}

func TestMethodNotAllowedListsAllowed(t *testing.T) {
	h := NewHandler(NewStore())
	rec := do(h, "PATCH", "/links/abc", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "DELETE, GET, HEAD, PUT" {
		t.Fatalf("Allow = %q", got)
	}
}

func TestPostCreatesNewResourceEachTime(t *testing.T) {
	h := NewHandler(NewStore())
	first := do(h, "POST", "/links", `{"target":"https://example.com/a"}`)
	second := do(h, "POST", "/links", `{"target":"https://example.com/a"}`)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("codes = %d, %d, want 201, 201", first.Code, second.Code)
	}
	a, b := first.Header().Get("Location"), second.Header().Get("Location")
	if !strings.HasPrefix(a, "/links/") || a == b {
		t.Fatalf("Locations = %q, %q, want two different /links/ URLs", a, b)
	}
	if got := do(h, "GET", a, "").Code; got != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", a, got)
	}
}

func TestPutIsIdempotentAndReplaces(t *testing.T) {
	h := NewHandler(NewStore())
	body := `{"target":"https://example.com/a","owner":"u1"}`
	if rec := do(h, "PUT", "/links/abc", body); rec.Code != 201 || rec.Header().Get("Location") != "/links/abc" {
		t.Fatalf("first PUT = %d, Location %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := do(h, "PUT", "/links/abc", body); rec.Code != 200 {
		t.Fatalf("second PUT = %d, want 200", rec.Code)
	}
	do(h, "PUT", "/links/abc", `{"target":"https://example.com/b"}`)
	var l Link
	if err := json.Unmarshal(do(h, "GET", "/links/abc", "").Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.Target != "https://example.com/b" || l.Owner != "" {
		t.Fatalf("link = %+v, want the whole resource replaced", l)
	}
}

func TestCodesAreNotReusedAfterDelete(t *testing.T) {
	h := NewHandler(NewStore())
	a := do(h, "POST", "/links", `{"target":"https://example.com/a"}`).Header().Get("Location")
	b := do(h, "POST", "/links", `{"target":"https://example.com/b"}`).Header().Get("Location")
	do(h, "DELETE", a, "")
	c := do(h, "POST", "/links", `{"target":"https://example.com/c"}`).Header().Get("Location")
	if c == a || c == b {
		t.Fatalf("new code %q collides with %q or %q", c, a, b)
	}
}

func TestInvalidBodies(t *testing.T) {
	h := NewHandler(NewStore())
	for name, body := range map[string]string{
		"not json":   "https://example.com",
		"no target":  `{}`,
		"bad scheme": `{"target":"ftp://example.com"}`,
	} {
		if got := do(h, "POST", "/links", body).Code; got != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, got)
		}
	}
	if got := do(h, "POST", "/links", `{"target":"https://e.com/`+strings.Repeat("a", 5000)+`"}`).Code; got != 413 {
		t.Errorf("big body: got %d, want 413", got)
	}
}

func TestListFilterAndPaging(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/a", `{"target":"https://example.com/1","owner":"u1"}`)
	do(h, "PUT", "/links/b", `{"target":"https://example.org/2","owner":"u1"}`)
	do(h, "PUT", "/links/c", `{"target":"https://example.com/3","owner":"u2"}`)

	if got := items(t, do(h, "GET", "/links", "")); len(got) != 3 {
		t.Fatalf("all = %d items, want 3", len(got))
	}
	if got := items(t, do(h, "GET", "/links?host=example.com", "")); len(got) != 2 {
		t.Fatalf("host filter = %d items, want 2", len(got))
	}
	if got := items(t, do(h, "GET", "/links?limit=1&offset=1", "")); len(got) != 1 || got[0].Code != "b" {
		t.Fatalf("page = %+v, want only b", got)
	}
	if got := items(t, do(h, "GET", "/users/u1/links", "")); len(got) != 2 {
		t.Fatalf("u1 links = %d, want 2", len(got))
	}
	rec := do(h, "GET", "/users/nobody/links", "")
	if got := items(t, rec); len(got) != 0 || !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("empty list = %s, want an empty array, not null", rec.Body)
	}
	for _, q := range []string{"limit=0", "limit=101", "limit=abc", "offset=-1"} {
		if got := do(h, "GET", "/links?"+q, "").Code; got != http.StatusBadRequest {
			t.Errorf("?%s = %d, want 400", q, got)
		}
	}
}

func TestDisableAction(t *testing.T) {
	h := NewHandler(NewStore())
	do(h, "PUT", "/links/abc", `{"target":"https://example.com/a"}`)
	if got := do(h, "GET", "/abc", "").Code; got != http.StatusFound {
		t.Fatalf("before disable: %d, want 302", got)
	}
	for range 2 {
		if got := do(h, "POST", "/links/abc/disable", "").Code; got != http.StatusOK {
			t.Fatalf("disable = %d, want 200 every time", got)
		}
	}
	if got := do(h, "GET", "/abc", "").Code; got != http.StatusGone {
		t.Fatalf("after disable: %d, want 410", got)
	}
}
