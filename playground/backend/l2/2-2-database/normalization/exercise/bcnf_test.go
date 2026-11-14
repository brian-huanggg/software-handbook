//go:build exercise

package exercise

import (
	"testing"

	norm "handbook/playground/backend/normalization"
)

// 每位審核員只負責一個網域；一條連結在一個網域上只有一位審核員。
var reviews = []norm.Row{
	{"link_id": "1", "domain": "a.example", "reviewer_id": "r1"},
	{"link_id": "2", "domain": "a.example", "reviewer_id": "r1"},
	{"link_id": "1", "domain": "b.example", "reviewer_id": "r2"},
	{"link_id": "3", "domain": "b.example", "reviewer_id": "r3"},
}

func TestIsSuperkey(t *testing.T) {
	tests := []struct {
		cols []string
		want bool
	}{
		{[]string{"link_id", "domain"}, true},
		{[]string{"link_id", "reviewer_id"}, true},
		{[]string{"link_id"}, false},
		{[]string{"reviewer_id"}, false},
	}
	for _, tt := range tests {
		if got := IsSuperkey(reviews, tt.cols); got != tt.want {
			t.Errorf("IsSuperkey(%v) = %v, want %v", tt.cols, got, tt.want)
		}
	}
}

func TestBCNFViolations(t *testing.T) {
	fds := []norm.FD{
		{LHS: []string{"link_id", "domain"}, RHS: []string{"reviewer_id"}},
		{LHS: []string{"reviewer_id"}, RHS: []string{"domain"}},
		{LHS: []string{"link_id", "domain"}, RHS: []string{"link_id"}}, // 平凡相依
	}
	got := BCNFViolations(reviews, fds)
	if len(got) != 1 || got[0].String() != "reviewer_id -> domain" {
		t.Fatalf("got %v, want only [reviewer_id -> domain]", got)
	}
}

// 資料推翻的相依不是相依，不該被列為違規。
func TestBCNFIgnoresFDsThatDoNotHold(t *testing.T) {
	fds := []norm.FD{{LHS: []string{"domain"}, RHS: []string{"reviewer_id"}}}
	if got := BCNFViolations(reviews, fds); len(got) != 0 {
		t.Fatalf("domain -> reviewer_id 在資料裡不成立，不該列入：%v", got)
	}
}
