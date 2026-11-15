//go:build exercise

// Package exercise: finish Sweep so a crash at any point leaves no residue.
package exercise

import "errors"

type Store interface {
	Delete(key string) error // idempotent: a missing object is not an error
}

type Queue interface {
	Pending() []string
	Remove(key string) error
}

// Sweep deletes each queued object, then removes its queue row.
// A failure on one key must not stop the others, and a row must stay
// in the queue until its object is confirmed gone. Return all errors joined.
func Sweep(s Store, q Queue) error {
	return errors.New("not implemented")
}
