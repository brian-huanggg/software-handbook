//go:build exercise

package exercise

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"testing"
)

func pixelBomb(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestRealPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())
	info, err := Inspect(r, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if info != (Info{Type: "image/png", Ext: ".png", Width: 4, Height: 3}) {
		t.Fatalf("got %+v", info)
	}
	if pos, _ := r.Seek(0, io.SeekCurrent); pos != 0 {
		t.Fatalf("reader left at offset %d, want 0", pos)
	}
}

func TestHTMLRejected(t *testing.T) {
	_, err := Inspect(bytes.NewReader([]byte("<html><script>x()</script>")), 1_000_000)
	if !errors.Is(err, ErrType) {
		t.Fatalf("want ErrType, got %v", err)
	}
}

func TestShortTextRejected(t *testing.T) {
	_, err := Inspect(bytes.NewReader([]byte("hi")), 1_000_000)
	if !errors.Is(err, ErrType) {
		t.Fatalf("want ErrType, got %v", err)
	}
}

func TestPixelBombRejected(t *testing.T) {
	_, err := Inspect(bytes.NewReader(pixelBomb(50000, 50000)), 40_000_000)
	if !errors.Is(err, ErrPixels) {
		t.Fatalf("want ErrPixels, got %v", err)
	}
}

func TestNoOverflow(t *testing.T) {
	// 2^31-1 squared overflows int32; the check must still reject it.
	_, err := Inspect(bytes.NewReader(pixelBomb(1<<31-1, 1<<31-1)), 40_000_000)
	if !errors.Is(err, ErrPixels) && !errors.Is(err, ErrType) {
		t.Fatalf("want rejection, got %v", err)
	}
}
