package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var errTooLarge = errors.New("file too large")

// DirStore keeps objects under Dir/objects. Uploads are first written to
// Dir/tmp on the same filesystem, so the final rename is atomic.
type DirStore struct{ Dir string }

func (s DirStore) tmpDir() string     { return filepath.Join(s.Dir, "tmp") }
func (s DirStore) objectsDir() string { return filepath.Join(s.Dir, "objects") }

func (s DirStore) init() error {
	for _, d := range []string{s.tmpDir(), s.objectsDir()} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return err
		}
	}
	return nil
}

// Stage streams r into a temp file and refuses anything over max bytes.
// On any error the temp file is already gone.
func (s DirStore) Stage(r io.Reader, max int64) (tmpPath string, err error) {
	f, err := os.CreateTemp(s.tmpDir(), "upload-*")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(f.Name())
		}
	}()
	// Read one byte past the limit so "exactly max" and "too big" differ.
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if n > max {
		_ = f.Close()
		return "", errTooLarge
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// Commit moves a staged file to a brand-new key. Nothing references the key
// yet, so a crash after this call leaves an orphan, not a dangling record.
func (s DirStore) Commit(tmpPath, key string) error {
	return os.Rename(tmpPath, filepath.Join(s.objectsDir(), key))
}

func (s DirStore) Remove(key string) error {
	return os.Remove(filepath.Join(s.objectsDir(), key))
}

// Discard deletes a staged file that will never be committed.
func (s DirStore) Discard(tmpPath string) {
	_ = os.Remove(tmpPath)
}

func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("new key: %w", err)
	}
	return hex.EncodeToString(b), nil
}
