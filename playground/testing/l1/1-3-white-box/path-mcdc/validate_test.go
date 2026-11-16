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

// 這四個案例滿足判定／條件覆蓋，但還沒達成 MC/DC。
var targetCases = []targetCase{
	{"ftp 被擋", "ftp", "a.com", ErrRejected},
	{"http 通過", "http", "a.com", nil},
	{"https 空 host 被擋", "https", "", ErrRejected},
	{"ftp 空 host 被擋", "ftp", "", ErrRejected},
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
