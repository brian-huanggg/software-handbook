package filecheck

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func smallPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pixelBomb is a 33-byte PNG header that claims w x h pixels.
func pixelBomb(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestInspectAcceptsRealPNG(t *testing.T) {
	info, err := Inspect(bytes.NewReader(smallPNG(t)), 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if info.Type != "image/png" || info.Ext != ".png" || info.Width != 4 || info.Height != 3 {
		t.Fatalf("got %+v", info)
	}
}

func TestInspectRejectsHTMLNamedPNG(t *testing.T) {
	// The client said avatar.png and image/png; the bytes say HTML.
	body := []byte("<html><script>fetch('/api/me')</script></html>")
	_, err := Inspect(bytes.NewReader(body), 1_000_000)
	if !errors.Is(err, ErrType) {
		t.Fatalf("want ErrType, got %v", err)
	}
}

func TestInspectRejectsPixelBomb(t *testing.T) {
	bomb := pixelBomb(50000, 50000)
	if len(bomb) > 64 {
		t.Fatalf("bomb is %d bytes, expected tiny", len(bomb))
	}
	_, err := Inspect(bytes.NewReader(bomb), 40_000_000)
	if !errors.Is(err, ErrPixels) {
		t.Fatalf("want ErrPixels, got %v", err)
	}
}

func TestDetectTypeGIFPolyglotPasses(t *testing.T) {
	// Magic bytes only prove the start looks like a GIF. This is why
	// serving with nosniff and a fixed Content-Type still matters.
	_, ext, err := DetectType([]byte("GIF89a/*<script>alert(1)</script>*/"))
	if err != nil || ext != ".gif" {
		t.Fatalf("sniffing is expected to accept this: ext=%q err=%v", ext, err)
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]string{
		"avatar.png":                      "avatar.png",
		"../../etc/cron.d/x":              "x",
		`C:\Users\amy\photo.jpg`:          "photo.jpg",
		"report\u202egnp.exe":             "reportgnp.exe",
		"a\x00b\n.png":                    "ab.png",
		"  ..  ":                          "file",
		"":                                "file",
		strings.Repeat("長", 100) + ".png": strings.Repeat("長", 85),
	}
	for in, want := range cases {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRootBlocksTraversal(t *testing.T) {
	dir := t.TempDir()
	// filepath.Join cleans the path, and cleaning keeps the escape.
	if got := filepath.Join(dir, "../../etc/x"); strings.HasPrefix(got, dir) {
		t.Fatalf("expected Join to escape, got %s", got)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.Create("../../etc/x"); err == nil {
		t.Fatal("os.Root allowed an escaping path")
	}
}

func makeZip(t *testing.T, files map[string][]byte) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}

var lim = Limits{MaxEntries: 10, MaxTotal: 1 << 20}

func TestExtractOK(t *testing.T) {
	dir := t.TempDir()
	zr := makeZip(t, map[string][]byte{"links/a.csv": []byte("code,url\n")})
	if err := Extract(zr, dir, lim); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "links", "a.csv")); err != nil {
		t.Fatal(err)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	zr := makeZip(t, map[string][]byte{"../../evil.sh": []byte("x")})
	if err := Extract(zr, t.TempDir(), lim); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("want ErrUnsafePath, got %v", err)
	}
}

func TestExtractStopsBomb(t *testing.T) {
	// 10 MB of zeros compresses to about 10 KB.
	zr := makeZip(t, map[string][]byte{"big.csv": make([]byte, 10<<20)})
	if zr.File[0].CompressedSize64 > 64<<10 {
		t.Fatalf("expected high compression, got %d", zr.File[0].CompressedSize64)
	}
	if err := Extract(zr, t.TempDir(), lim); !errors.Is(err, ErrExpanded) {
		t.Fatalf("want ErrExpanded, got %v", err)
	}
}

func TestExtractRejectsTooManyEntries(t *testing.T) {
	files := map[string][]byte{}
	for i := range 11 {
		files[filepath.Join("f", string(rune('a'+i)))] = nil
	}
	if err := Extract(makeZip(t, files), t.TempDir(), lim); !errors.Is(err, ErrTooMany) {
		t.Fatalf("want ErrTooMany, got %v", err)
	}
}

func TestServeStoredSetsNosniff(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := ServeStored(rec, Info{Type: "image/png"}, strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("headers: %v", rec.Header())
	}
}
