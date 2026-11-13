// Package exercise holds the starter code for the certification exercise.
package exercise

import "errors"

var ErrConflict = errors.New("deadlock found when trying to get lock; try restarting transaction")

// Certify reports ErrConflict when any key in writes was committed by
// another transaction after snapshot. index maps a key to the seqno of
// its last committed writer.
func Certify(index map[string]uint64, snapshot uint64, writes map[string]string) error {
	// TODO: implement.
	return nil
}
