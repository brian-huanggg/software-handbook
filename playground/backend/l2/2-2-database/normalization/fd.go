// Package normalization 用幾十行程式把「函數相依」變成可以檢查的東西。
package normalization

import "strings"

// Row 是一列資料：欄位名稱對應值。
type Row map[string]string

// FD 表示函數相依 LHS → RHS：同樣的 LHS 值，一定對應同樣的 RHS 值。
type FD struct {
	LHS []string
	RHS []string
}

func (f FD) String() string {
	return strings.Join(f.LHS, ",") + " -> " + strings.Join(f.RHS, ",")
}

// project 取出指定欄位，組成可以當 map key 的字串。
// 用 \x00 分隔，避免 ("a","bc") 與 ("ab","c") 被當成同一組值。
func project(r Row, cols []string) string {
	vals := make([]string, len(cols))
	for i, c := range cols {
		vals[i] = r[c]
	}
	return strings.Join(vals, "\x00")
}

// HoldsFD 檢查 fd 在這批資料裡是否成立。
// 注意：資料只能證明相依「被推翻」，不能證明它永遠成立；
// 真正的相依來自業務規則，不是來自目前剛好長這樣的資料。
func HoldsFD(rows []Row, fd FD) bool {
	seen := make(map[string]string, len(rows))
	for _, r := range rows {
		l, rhs := project(r, fd.LHS), project(r, fd.RHS)
		if prev, ok := seen[l]; ok && prev != rhs {
			return false
		}
		seen[l] = rhs
	}
	return true
}
