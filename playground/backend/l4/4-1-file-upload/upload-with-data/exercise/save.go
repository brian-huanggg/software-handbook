//go:build exercise

// Package exercise: finish Save so a file and its record never disagree.
package exercise

import (
	"context"
	"errors"
)

type Store interface {
	Commit(tmpPath, key string) error // move a staged file to a new key
	Remove(key string) error
}

type Repo interface {
	Create(ctx context.Context, previewKey string) error
}

// Save publishes the staged file at tmpPath under key and records it.
//
// Rules:
//  1. Move the file first, then insert the record.
//  2. If the move fails, return its error and do not touch the repo.
//  3. If the insert fails, remove the moved file and return the insert error.
func Save(ctx context.Context, s Store, r Repo, tmpPath, key string) error {
	return errors.New("not implemented")
}
