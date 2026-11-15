package consistency

import (
	"errors"
	"sort"
	"sync"
)

// DB is an in-memory stand-in for a relational database with one table of
// avatars (user -> object key) and one table pending_deletions. Update runs
// a function as a transaction: all writes apply on commit, or none do.
type DB struct {
	mu      sync.Mutex
	avatars map[string]string
	pending map[string]struct{}

	// FailCommit simulates a crash or timeout at commit time.
	FailCommit func() error
}

func NewDB() *DB {
	return &DB{avatars: map[string]string{}, pending: map[string]struct{}{}}
}

// Tx is the view of the database inside one transaction.
type Tx struct {
	avatars map[string]string
	pending map[string]struct{}
}

func (t *Tx) Avatar(user string) string        { return t.avatars[user] }
func (t *Tx) SetAvatar(user, key string)       { t.avatars[user] = key }
func (t *Tx) AddPendingDeletion(key string)    { t.pending[key] = struct{}{} }
func (t *Tx) RemovePendingDeletion(key string) { delete(t.pending, key) }

var ErrCommit = errors.New("commit failed")

func (d *DB) Update(fn func(*Tx) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx := &Tx{avatars: clone(d.avatars), pending: clone(d.pending)}
	if err := fn(tx); err != nil {
		return err // rollback: the copy is simply dropped
	}
	if d.FailCommit != nil {
		if err := d.FailCommit(); err != nil {
			return errors.Join(ErrCommit, err)
		}
	}
	d.avatars, d.pending = tx.avatars, tx.pending
	return nil
}

func (d *DB) Avatar(user string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.avatars[user]
}

func (d *DB) Avatars() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return sortedValues(d.avatars)
}

func (d *DB) PendingDeletions() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	keys := make([]string, 0, len(d.pending))
	for k := range d.pending {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func clone[K comparable, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func sortedValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
