// Package filecheck decides what an uploaded file is from its bytes,
// never from the name or headers the client sent.
package filecheck

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif" // register decoders for image.DecodeConfig
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
)

var (
	ErrType   = errors.New("file type not allowed")
	ErrPixels = errors.New("image dimensions too large")
)

// allowed maps a sniffed type to the extension the server will use.
// The client's extension is never consulted.
var allowed = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
}

// Info is what the server decided about a file.
type Info struct {
	Type          string // sniffed MIME type, e.g. "image/png"
	Ext           string // extension chosen by the server
	Width, Height int
}

// DetectType sniffs the first bytes of a file and checks the allowlist.
func DetectType(head []byte) (mime, ext string, err error) {
	mime = http.DetectContentType(head) // looks at no more than 512 bytes
	ext, ok := allowed[mime]
	if !ok {
		return mime, "", fmt.Errorf("%w: %s", ErrType, mime)
	}
	return mime, ext, nil
}

// Inspect reads the head of f to learn its type, then reads only the
// image header to learn its size. It never decodes pixels.
func Inspect(f io.ReadSeeker, maxPixels int64) (Info, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Info{}, err
	}
	mime, ext, err := DetectType(head[:n])
	if err != nil {
		return Info{}, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	// DecodeConfig parses only the header: a 40 KB file can claim
	// 50000x50000 pixels, and we want to know before allocating 10 GB.
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Info{}, fmt.Errorf("%w: unreadable image header: %v", ErrType, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return Info{}, fmt.Errorf("%w: %dx%d", ErrPixels, cfg.Width, cfg.Height)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	return Info{Type: mime, Ext: ext, Width: cfg.Width, Height: cfg.Height}, nil
}
