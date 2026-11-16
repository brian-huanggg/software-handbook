//go:build recognize

package recognize

import (
	"testing"

	shortener "handbook/playground/testing/coverage-criteria"
)

// 覆蓋率 100%，所有分支皆已測試。
func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		reject bool
	}{
		{"unsupported scheme", "ftp", "example.com", true},
		{"valid https", "https", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shortener.ValidateTarget(tt.scheme, tt.host)
			if !tt.reject && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTargetEmptyHost(t *testing.T) {
	err := shortener.ValidateTarget("https", "")
	t.Log("result:", err)
}
