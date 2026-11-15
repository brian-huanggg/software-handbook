//go:build recognize

// Package recognize is AI-generated code for the review exercise.
package recognize

import (
	"io"
	"time"

	consistency "handbook/playground/backend/cross-system-consistency"
)

type Service struct {
	Store consistency.ObjectStore
	DB    *consistency.DB
}

func (s *Service) ReplaceAvatar(user string, content io.Reader) error {
	key := "avatars/" + user + ".png"
	old := s.DB.Avatar(user)

	if old != "" {
		if err := s.Store.Delete(old); err != nil {
			return err
		}
	}
	if err := s.Store.Put(key, content); err != nil {
		return err
	}
	return s.DB.Update(func(tx *consistency.Tx) error {
		tx.SetAvatar(user, key)
		return nil
	})
}

// Cleanup removes objects that no avatar refers to.
func (s *Service) Cleanup() error {
	objs, err := s.Store.List()
	if err != nil {
		return err
	}
	used := map[string]bool{}
	for _, k := range s.DB.Avatars() {
		used[k] = true
	}
	for _, o := range objs {
		if !used[o.Key] && time.Since(o.ModTime) > 0 {
			s.Store.Delete(o.Key)
		}
	}
	return nil
}
