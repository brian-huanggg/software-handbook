package consistency

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
)

type Service struct {
	Store ObjectStore
	DB    *DB
}

// newKey returns a key nobody has used before, so an upload never
// overwrites an object some record still points at.
func newKey(user string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("avatars/%s/%s", user, hex.EncodeToString(b[:])), nil
}

// ReplaceAvatar swaps a user's avatar. Order matters:
//  1. write the new object under a fresh key
//  2. in ONE transaction: point the user at it, queue the old key for deletion
//  3. the sweeper deletes queued objects later
func (s *Service) ReplaceAvatar(user string, content io.Reader) (string, error) {
	key, err := newKey(user)
	if err != nil {
		return "", err
	}
	if err := s.Store.Put(key, content); err != nil {
		return "", fmt.Errorf("put new object: %w", err)
	}

	err = s.DB.Update(func(tx *Tx) error {
		if old := tx.Avatar(user); old != "" {
			tx.AddPendingDeletion(old) // same transaction as the pointer change
		}
		tx.SetAvatar(user, key)
		return nil
	})
	if err != nil {
		// Best effort. If this fails too, the new object is an orphan
		// that the reconciliation scan will find.
		if delErr := s.Store.Delete(key); delErr != nil {
			log.Printf("orphan object %s: %v", key, delErr)
		}
		return "", fmt.Errorf("update record: %w", err)
	}
	return key, nil
}
