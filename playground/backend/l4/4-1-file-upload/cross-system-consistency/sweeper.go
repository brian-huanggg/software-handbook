package consistency

import (
	"errors"
	"fmt"
	"time"
)

// Sweep deletes every object queued in pending_deletions. It is safe to run
// any number of times, concurrently with uploads, and to rerun after a crash:
// a row is removed only after its object is confirmed gone.
func (s *Service) Sweep() error {
	var errs []error
	for _, key := range s.DB.PendingDeletions() {
		if err := s.Store.Delete(key); err != nil {
			errs = append(errs, fmt.Errorf("delete %s: %w", key, err))
			continue // leave the row; the next run retries
		}
		err := s.DB.Update(func(tx *Tx) error {
			tx.RemovePendingDeletion(key)
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Orphans lists objects that no avatar and no pending row refers to and
// that are older than grace. The grace period protects an upload whose
// transaction has not committed yet.
func (s *Service) Orphans(now time.Time, grace time.Duration) ([]string, error) {
	objs, err := s.Store.List()
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, k := range s.DB.Avatars() {
		known[k] = true
	}
	for _, k := range s.DB.PendingDeletions() {
		known[k] = true
	}
	var out []string
	for _, o := range objs {
		if !known[o.Key] && now.Sub(o.ModTime) > grace {
			out = append(out, o.Key)
		}
	}
	return out, nil
}
