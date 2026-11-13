//go:build exercise

package exercise

import (
	"time"

	flags "handbook/playground/backend/feature-flags"
)

// Stale 回傳已過期（expires 早於 now 的日期）的旗標名稱，依字母排序。
// 沒有 expires 的旗標也要列出：沒有到期日的旗標，永遠沒人記得清。
func Stale(set flags.Set, now time.Time) []string {
	// TODO: 實作
	return nil
}
