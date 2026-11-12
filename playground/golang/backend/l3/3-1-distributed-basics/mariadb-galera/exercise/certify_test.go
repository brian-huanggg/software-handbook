//go:build exercise

package exercise

import (
	"errors"
	"testing"
)

func TestCertify(t *testing.T) {
	index := map[string]uint64{"abc": 7, "xyz": 3}

	tests := []struct {
		name     string
		snapshot uint64
		writes   map[string]string
		want     error
	}{
		{"untouched key", 7, map[string]string{"new": "u"}, nil},
		{"writer is inside snapshot", 7, map[string]string{"abc": "u"}, nil},
		{"writer is after snapshot", 6, map[string]string{"abc": "u"}, ErrConflict},
		{"one conflict among many keys", 5, map[string]string{"xyz": "u", "abc": "u"}, ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Certify(index, tt.snapshot, tt.writes)
			if !errors.Is(got, tt.want) {
				t.Fatalf("Certify() = %v, want %v", got, tt.want)
			}
		})
	}
}
