//go:build exercise

package shortener

import "testing"

// Trace 記錄每個原子條件在一次呼叫裡「實際被求值」的結果。
// nil 表示被短路跳過，沒有求值。
type Trace struct {
	HTTP  *bool // scheme == "http"
	HTTPS *bool // scheme == "https"
	Empty *bool // host == ""
}

func ptr(b bool) *bool { return &b }

// TraceValidate 與 ValidateTarget 的判斷完全一樣，只是把每個原子條件的求值結果記下來。
func TraceValidate(scheme, host string) Trace {
	var tr Trace
	tr.HTTP = ptr(scheme == "http")
	if !*tr.HTTP {
		tr.HTTPS = ptr(scheme == "https")
	}
	if !(*tr.HTTP || (tr.HTTPS != nil && *tr.HTTPS)) {
		return tr // 前半段已經為真，|| 短路，host 不會被檢查
	}
	tr.Empty = ptr(host == "")
	return tr
}

// TestConditionCoverage 重播 TestValidateTarget 的所有案例，
// 檢查每個原子條件都至少被求值為真一次、為假一次。
func TestConditionCoverage(t *testing.T) {
	seen := map[string][2]bool{} // 條件名 -> [見過假, 見過真]
	mark := func(name string, v *bool) {
		if v == nil {
			return
		}
		s := seen[name]
		if *v {
			s[1] = true
		} else {
			s[0] = true
		}
		seen[name] = s
	}
	for _, c := range targetCases {
		tr := TraceValidate(c.scheme, c.host)
		mark(`scheme == "http"`, tr.HTTP)
		mark(`scheme == "https"`, tr.HTTPS)
		mark(`host == ""`, tr.Empty)
	}
	for _, name := range []string{`scheme == "http"`, `scheme == "https"`, `host == ""`} {
		s := seen[name]
		if !s[0] {
			t.Errorf("條件 %s 從沒被求值為假", name)
		}
		if !s[1] {
			t.Errorf("條件 %s 從沒被求值為真", name)
		}
	}
}
