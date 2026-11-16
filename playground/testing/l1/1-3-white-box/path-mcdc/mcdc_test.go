package shortener

import "testing"

// conditionNames 依序是 ValidateTarget 判斷式裡的三個原子條件 P、Q、R。
var conditionNames = []string{`scheme == "http"`, `scheme == "https"`, `host == ""`}

// conditions 算出一個案例裡三個原子條件的值。
// 這裡用 unique-cause MC/DC：條件的值直接由輸入決定，不管有沒有被短路跳過。
func conditions(c targetCase) [3]bool {
	return [3]bool{c.scheme == "http", c.scheme == "https", c.host == ""}
}

// independencePair 為第 i 個條件找一組獨立影響對：
// 兩個案例只有第 i 個條件不同，其他條件相同，而且判斷結果不同。
// 找不到時 ok 為 false。
func independencePair(cases []targetCase, i int) (a, b targetCase, ok bool) {
	for x := range cases {
		for y := x + 1; y < len(cases); y++ {
			cx, cy := conditions(cases[x]), conditions(cases[y])
			if cx[i] == cy[i] {
				continue
			}
			same := true
			for j := range cx {
				if j != i && cx[j] != cy[j] {
					same = false
				}
			}
			rejectX := ValidateTarget(cases[x].scheme, cases[x].host) != nil
			rejectY := ValidateTarget(cases[y].scheme, cases[y].host) != nil
			if same && rejectX != rejectY {
				return cases[x], cases[y], true
			}
		}
	}
	return targetCase{}, targetCase{}, false
}

// 用一組已知達成 MC/DC 的案例確認 independencePair 本身是對的。
func TestIndependencePairFinder(t *testing.T) {
	known := []targetCase{
		{"ftp", "ftp", "a.com", ErrRejected},
		{"http", "http", "a.com", nil},
		{"https", "https", "a.com", nil},
		{"https 空 host", "https", "", ErrRejected},
	}
	for i, name := range conditionNames {
		if _, _, ok := independencePair(known, i); !ok {
			t.Errorf("條件 %s 應該找得到獨立影響對", name)
		}
	}
}
