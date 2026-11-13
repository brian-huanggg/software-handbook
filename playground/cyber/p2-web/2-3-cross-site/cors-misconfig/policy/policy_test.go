package policy

import "testing"

const app = "https://app.sho.rt"

// Origins an attacker can actually produce: a page on their own host, a
// sandboxed iframe, and two domains they could register.
var attackers = []string{
	"http://127.0.0.1:9666",
	"null",
	"https://evilsho.rt",
	"https://app.sho.rt.evil.example",
}

func TestChecks(t *testing.T) {
	checks := map[string]Check{
		"reflect":   Reflect(),
		"null":      AllowNull(app),
		"suffix":    Suffix("sho.rt"),
		"prefix":    Prefix(app),
		"allowlist": Allowlist(app),
	}
	// Which attacker origins each check lets through.
	want := map[string][]string{
		"reflect":   attackers,
		"null":      {"null"},
		"suffix":    {"https://evilsho.rt"},
		"prefix":    {"https://app.sho.rt.evil.example"},
		"allowlist": nil,
	}
	for name, check := range checks {
		if !check(app) {
			t.Errorf("%s: legitimate origin %s refused", name, app)
		}
		pass := map[string]bool{}
		for _, o := range want[name] {
			pass[o] = true
		}
		for _, o := range attackers {
			if got := check(o); got != pass[o] {
				t.Errorf("%s(%q) = %v, want %v", name, o, got, pass[o])
			}
		}
	}
}
