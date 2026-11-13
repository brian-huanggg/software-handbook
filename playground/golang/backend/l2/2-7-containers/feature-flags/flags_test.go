package flags

import (
	"fmt"
	"testing"
)

func load(t *testing.T, s *Store, js string) {
	t.Helper()
	if err := s.Load([]byte(js)); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestUnknownFlagIsOff(t *testing.T) {
	var s Store
	if s.Enabled("nope", "u1") {
		t.Fatal("unknown flag must be off")
	}
}

func TestStickyPerUser(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":true,"percent":50}}`)
	for i := range 200 {
		u := fmt.Sprintf("u%d", i)
		first := s.Enabled("f", u)
		for range 5 {
			if s.Enabled("f", u) != first {
				t.Fatalf("user %s flipped between calls", u)
			}
		}
	}
}

func TestRampIsMonotonic(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":true,"percent":10}}`)
	var in10 []string
	for i := range 5000 {
		u := fmt.Sprintf("u%d", i)
		if s.Enabled("f", u) {
			in10 = append(in10, u)
		}
	}
	load(t, &s, `{"f":{"enabled":true,"percent":20}}`)
	for _, u := range in10 {
		if !s.Enabled("f", u) {
			t.Fatalf("user %s was in 10%% but dropped out at 20%%", u)
		}
	}
}

func TestPercentRoughlyMatches(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":true,"percent":10}}`)
	n, hit := 20000, 0
	for i := range n {
		if s.Enabled("f", fmt.Sprintf("user-%d", i)) {
			hit++
		}
	}
	if got := float64(hit) / float64(n); got < 0.08 || got > 0.12 {
		t.Fatalf("ratio = %.3f, want about 0.10", got)
	}
}

func TestKillSwitchBeatsEverything(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":false,"percent":100,"allow_users":["boss"]}}`)
	if s.Enabled("f", "boss") || s.Enabled("f", "u1") {
		t.Fatal("enabled=false must turn the flag off for everyone")
	}
}

func TestAllowUsers(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":true,"percent":0,"allow_users":["alice"]}}`)
	if !s.Enabled("f", "alice") || s.Enabled("f", "bob") {
		t.Fatal("only alice should be on")
	}
}

func TestInvalidConfigKeepsOld(t *testing.T) {
	var s Store
	load(t, &s, `{"f":{"enabled":true,"percent":100}}`)
	for _, bad := range []string{
		`{not json`,
		`{"f":{"enabled":true,"percent":101}}`,
		`{"f":{"enabled":true,"percent":-1}}`,
		`{"f":{"enabled":true,"percent":5,"expires":"next week"}}`,
	} {
		if err := s.Load([]byte(bad)); err == nil {
			t.Fatalf("Load(%s) should fail", bad)
		}
		if !s.Enabled("f", "u1") {
			t.Fatalf("old config lost after bad input %s", bad)
		}
	}
}

func TestFlagsAreIndependent(t *testing.T) {
	var s Store
	load(t, &s, `{"a":{"enabled":true,"percent":30},"b":{"enabled":true,"percent":30}}`)
	same := 0
	for i := range 2000 {
		u := fmt.Sprintf("u%d", i)
		if s.Enabled("a", u) == s.Enabled("b", u) {
			same++
		}
	}
	// 若兩個旗標共用同一組分桶，same 會是 2000。
	if same == 2000 {
		t.Fatal("flags share the same cohort")
	}
}
