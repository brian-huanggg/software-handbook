// Package consistency shows how a database and an object store stay
// consistent without a shared transaction.
package consistency

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ObjectInfo is what a listing returns for one object.
type ObjectInfo struct {
	Key     string
	ModTime time.Time
}

// ObjectStore is the small slice of an S3-like API this article needs.
type ObjectStore interface {
	Put(key string, r io.Reader) error
	// Delete must succeed when the object is already gone (idempotent).
	Delete(key string) error
	Exists(key string) bool
	List() ([]ObjectInfo, error)
}

// DirStore keeps objects as files under Root. The hooks let tests inject
// failures at the exact step they want to study.
type DirStore struct {
	Root       string
	FailPut    func(key string) error
	FailDelete func(key string) error
}

func (s *DirStore) Put(key string, r io.Reader) error {
	if s.FailPut != nil {
		if err := s.FailPut(key); err != nil {
			return err
		}
	}
	path := filepath.Join(s.Root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// O_EXCL: a key is written once and never overwritten.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func (s *DirStore) Delete(key string) error {
	if s.FailDelete != nil {
		if err := s.FailDelete(key); err != nil {
			return err
		}
	}
	err := os.Remove(filepath.Join(s.Root, filepath.FromSlash(key)))
	if errors.Is(err, os.ErrNotExist) {
		return nil // already gone counts as success
	}
	return err
}

func (s *DirStore) Exists(key string) bool {
	_, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(key)))
	return err == nil
}

func (s *DirStore) List() ([]ObjectInfo, error) {
	var out []ObjectInfo
	err := filepath.WalkDir(s.Root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(s.Root, p)
		if err != nil {
			return err
		}
		out = append(out, ObjectInfo{Key: filepath.ToSlash(rel), ModTime: info.ModTime()})
		return nil
	})
	return out, err
}
