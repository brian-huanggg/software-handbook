//go:build recognize

package recognize

import (
	"errors"
	"testing"

	shortener "handbook/playground/testing/path-mcdc"
)

// 判斷式有 3 個條件，MC/DC 需要 N+1 = 4 個案例，以下 4 個案例達成 MC/DC。
//
//	scheme == "http"  的獨立影響：案例 1 與案例 2
//	scheme == "https" 的獨立影響：案例 3 與案例 2
//	host == ""        的獨立影響：案例 3 與案例 4
func TestValidateTargetMCDC(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		reject bool
	}{
		{"1 http ok", "http", "example.com", false},
		{"2 ftp empty host", "ftp", "", true},
		{"3 https ok", "https", "example.com", false},
		{"4 https empty host", "https", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shortener.ValidateTarget(tt.scheme, tt.host)
			if tt.reject != errors.Is(err, shortener.ErrRejected) {
				t.Errorf("ValidateTarget(%q, %q) = %v, reject = %v", tt.scheme, tt.host, err, tt.reject)
			}
			if !tt.reject && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
