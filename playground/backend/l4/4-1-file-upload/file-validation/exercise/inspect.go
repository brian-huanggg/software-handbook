//go:build exercise

// Package exercise: finish Inspect so the server, not the client,
// decides what an uploaded image is and how big it really is.
package exercise

import (
	"errors"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

var (
	ErrType   = errors.New("file type not allowed")
	ErrPixels = errors.New("image dimensions too large")
)

type Info struct {
	Type          string // sniffed MIME type
	Ext           string // extension chosen by the server: .png, .jpg or .gif
	Width, Height int
}

// Inspect must:
//  1. sniff the first 512 bytes (files may be shorter) with http.DetectContentType;
//  2. accept only image/png, image/jpeg and image/gif, else wrap ErrType;
//  3. read the image header with image.DecodeConfig, never image.Decode;
//  4. reject width*height > maxPixels with ErrPixels, without int overflow;
//  5. leave f positioned at offset 0 on success, so the caller can store it.
func Inspect(f io.ReadSeeker, maxPixels int64) (Info, error) {
	return Info{}, errors.New("TODO")
}
