//go:build exercise

package exercise

import (
	"errors"

	norm "handbook/playground/backend/normalization"
)

// IsSuperkey 回報 cols 是否是這批資料的超鍵：任兩列在 cols 上的值都不同。
func IsSuperkey(rows []norm.Row, cols []string) bool {
	panic(errors.New("not implemented"))
}

// BCNFViolations 回傳 fds 中違反 BCNF 的相依：
// 非平凡（RHS 不是 LHS 的子集），而且 LHS 不是超鍵。
func BCNFViolations(rows []norm.Row, fds []norm.FD) []norm.FD {
	panic(errors.New("not implemented"))
}
