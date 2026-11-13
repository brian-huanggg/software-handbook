package origin

import (
	"errors"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	const base = "https://sho.rt/admin"
	tests := []struct {
		name  string
		other string
		want  bool
	}{
		{"different path and query", "https://sho.rt/api/links?page=2", true},
		{"explicit default port", "https://sho.rt:443/", true},
		{"host case differs", "https://SHO.RT/", true},
		{"different scheme", "http://sho.rt/admin", false},
		{"different port", "https://sho.rt:8443/admin", false},
		{"subdomain", "https://api.sho.rt/admin", false},
		{"parent domain trick", "https://sho.rt.evil.example/admin", false},
	}
	a, err := Parse(base)
	if err != nil {
		t.Fatalf("Parse(%q): %v", base, err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Parse(tt.other)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.other, err)
			}
			if got := a.SameOrigin(b); got != tt.want {
				t.Errorf("SameOrigin(%s, %s) = %v, want %v", a, b, got, tt.want)
			}
		})
	}
}

// localhost and 127.0.0.1 reach the same process, but the browser compares
// strings, not the machine behind them.
func TestLocalhostIsNotLoopbackIP(t *testing.T) {
	a, err := Parse("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse("http://127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	if a.SameOrigin(b) {
		t.Errorf("%s and %s should be different origins", a, b)
	}
}

func TestString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"https://sho.rt:443/admin?x=1", "https://sho.rt"},
		{"http://127.0.0.1:8080/api/links", "http://127.0.0.1:8080"},
		{"http://[::1]:9090/", "http://[::1]:9090"},
	}
	for _, tt := range tests {
		o, err := Parse(tt.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if got := o.String(); got != tt.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOpaque(t *testing.T) {
	for _, in := range []string{"file:///etc/hosts", "data:text/html,hi"} {
		if _, err := Parse(in); !errors.Is(err, ErrOpaque) {
			t.Errorf("Parse(%q) error = %v, want ErrOpaque", in, err)
		}
	}
}
