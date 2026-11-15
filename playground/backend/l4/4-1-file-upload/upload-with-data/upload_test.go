package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type part struct{ name, value string }

// body builds a multipart body in the given order; "preview" becomes a file.
func body(t *testing.T, fileSize int, parts ...part) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		if p.name == "preview" {
			w, err := mw.CreateFormFile("preview", "a.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(bytes.Repeat([]byte("x"), fileSize)); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := mw.WriteField(p.name, p.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func setup(t *testing.T) (*Handler, *MemRepo, string) {
	t.Helper()
	dir := t.TempDir()
	store := DirStore{Dir: dir}
	if err := store.init(); err != nil {
		t.Fatal(err)
	}
	repo := &MemRepo{}
	return &Handler{Store: store, Repo: repo, MaxBody: 64 << 10, MaxFile: 32 << 10, MaxField: 256}, repo, dir
}

func post(h *Handler, b io.Reader, ct string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/links", b)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

var good = []part{{"target_url", "https://example.com/a"}, {"title", "hi"}, {"preview", ""}}

func TestSuccess(t *testing.T) {
	h, repo, dir := setup(t)
	b, ct := body(t, 1000, good...)
	if rec := post(h, b, ct); rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	links := repo.All()
	if len(links) != 1 || links[0].Title != "hi" {
		t.Fatalf("links = %+v", links)
	}
	got := files(t, dir)
	if len(got) != 1 || filepath.Base(got[0]) != links[0].PreviewKey {
		t.Fatalf("files = %v, want exactly the object for %s", got, links[0].PreviewKey)
	}
}

// countingReader tells us how much of the body the server really read.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

func TestBadFieldRejectedBeforeFileIsRead(t *testing.T) {
	h, repo, dir := setup(t)
	h.MaxBody = 4 << 20
	h.MaxFile = 4 << 20
	total := 2 << 20
	b, ct := body(t, total, part{"target_url", "not a url"}, part{"preview", ""})
	cr := &countingReader{r: b}
	rec := post(h, cr, ct)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	if cr.n > total/10 {
		t.Fatalf("server read %d of %d bytes before rejecting", cr.n, total)
	}
	if len(repo.All()) != 0 || len(files(t, dir)) != 0 {
		t.Fatal("nothing should be stored")
	}
}

func TestFileTooLarge(t *testing.T) {
	h, repo, dir := setup(t)
	b, ct := body(t, int(h.MaxFile)+1, good...)
	if rec := post(h, b, ct); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(repo.All()) != 0 || len(files(t, dir)) != 0 {
		t.Fatalf("leftovers: %v", files(t, dir))
	}
}

func TestBodyTooLarge(t *testing.T) {
	h, repo, dir := setup(t)
	h.MaxFile = 1 << 20 // file limit alone would allow it; the body cap must stop it
	b, ct := body(t, int(h.MaxBody)+1, good...)
	if rec := post(h, b, ct); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(repo.All()) != 0 || len(files(t, dir)) != 0 {
		t.Fatalf("leftovers: %v", files(t, dir))
	}
}

func TestFieldAfterFile(t *testing.T) {
	h, _, dir := setup(t)
	b, ct := body(t, 100, part{"target_url", "https://example.com"}, part{"preview", ""}, part{"title", "late"})
	if rec := post(h, b, ct); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(files(t, dir)) != 0 {
		t.Fatalf("leftovers: %v", files(t, dir))
	}
}

func TestMissingFile(t *testing.T) {
	h, _, _ := setup(t)
	b, ct := body(t, 0, part{"target_url", "https://example.com"})
	if rec := post(h, b, ct); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestInsertFailureRemovesObject(t *testing.T) {
	h, repo, dir := setup(t)
	repo.Err = errors.New("db down")
	b, ct := body(t, 100, good...)
	if rec := post(h, b, ct); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("orphan files: %v", got)
	}
}

func TestNotMultipart(t *testing.T) {
	h, _, _ := setup(t)
	if rec := post(h, strings.NewReader("{}"), "application/json"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
