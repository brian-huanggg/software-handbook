//go:build exercise

package exercise

import "testing"

func TestTypeAllowed(t *testing.T) {
	allow := []string{"text/*", "application/json", "image/svg+xml"}
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{"exact", "application/json", true},
		{"with parameter", "text/html; charset=utf-8", true},
		{"family wildcard", "text/css", true},
		{"upper case", "Application/JSON", true},
		{"suffix type is exact only", "image/svg+xml", true},
		{"png", "image/png", false},
		{"prefix is not a match", "application/jsonp", false},
		{"family needs the slash", "textual/plain", false},
		{"empty", "", false},
		{"malformed", "json", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TypeAllowed(tt.ct, allow); got != tt.want {
				t.Errorf("TypeAllowed(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}
