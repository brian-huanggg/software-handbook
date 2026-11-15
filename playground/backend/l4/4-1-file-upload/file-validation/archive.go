package filecheck

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrUnsafePath = errors.New("archive entry escapes target directory")
	ErrTooMany    = errors.New("archive has too many entries")
	ErrExpanded   = errors.New("archive expands beyond limit")
)

// Limits bound what an archive may cost us, whatever its headers claim.
type Limits struct {
	MaxEntries int
	MaxTotal   int64 // total bytes after decompression
}

// Extract unpacks zr into dir. Every limit is enforced on bytes actually
// written, because the sizes in zip headers are written by the uploader.
func Extract(zr *zip.Reader, dir string, lim Limits) error {
	if len(zr.File) > lim.MaxEntries {
		return fmt.Errorf("%w: %d", ErrTooMany, len(zr.File))
	}
	root, err := os.OpenRoot(dir) // all opens below stay inside dir
	if err != nil {
		return err
	}
	defer root.Close()

	budget := lim.MaxTotal
	for _, zf := range zr.File {
		if !filepath.IsLocal(zf.Name) {
			return fmt.Errorf("%w: %q", ErrUnsafePath, zf.Name)
		}
		if zf.FileInfo().IsDir() {
			continue
		}
		n, err := extractOne(root, zf, budget)
		if err != nil {
			return err
		}
		budget -= n
	}
	return nil
}

func extractOne(root *os.Root, zf *zip.File, budget int64) (int64, error) {
	if err := root.MkdirAll(filepath.Dir(zf.Name), 0o750); err != nil {
		return 0, err
	}
	src, err := zf.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()
	dst, err := root.Create(zf.Name)
	if err != nil {
		return 0, err
	}
	// Read one byte past the budget so "exactly at limit" and "over" differ.
	n, err := io.Copy(dst, io.LimitReader(src, budget+1))
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return n, err
	}
	if n > budget {
		return n, ErrExpanded
	}
	return n, nil
}
