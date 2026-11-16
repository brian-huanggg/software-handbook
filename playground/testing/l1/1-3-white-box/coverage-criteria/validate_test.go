package shortener

import (
	"errors"
	"testing"
)

type targetCase struct {
	name         string
	scheme, host string
	wantErr      error
}

var targetCases = []targetCase{
	{"ftp 被擋", "ftp", "a.com", ErrRejected},
	{"https 通過", "https", "a.com", nil},
}

func TestValidateTarget(t *testing.T) {
	for _, tc := range targetCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateTarget(tc.scheme, tc.host)
			if !errors.Is(got, tc.wantErr) {
				t.Fatalf("ValidateTarget(%q, %q) = %v，期望 %v", tc.scheme, tc.host, got, tc.wantErr)
			}
		})
	}
}
