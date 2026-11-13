// Package flags 是一個最小的功能旗標函式庫：穩定分桶、漸進放量、kill switch、熱重載。
package flags

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"slices"
	"sync/atomic"
	"time"
)

// Rule 描述一個旗標。Enabled 是總開關（kill switch）：false 時，其他欄位一律不生效。
type Rule struct {
	Enabled    bool     `json:"enabled"`
	Percent    int      `json:"percent"`     // 0–100，放量比例
	AllowUsers []string `json:"allow_users"` // 一律開啟的使用者，例如內部員工
	Expires    string   `json:"expires"`     // YYYY-MM-DD，提醒清理的日期
}

// Set 是旗標名稱到規則的對照。
type Set map[string]Rule

// Store 以原子指標持有目前生效的 Set，讀取不需要鎖。
type Store struct {
	cur atomic.Pointer[Set]
}

// Load 解析並驗證 JSON，全部通過才換上。驗證失敗時保留舊設定。
func (s *Store) Load(data []byte) error {
	var next Set
	if err := json.Unmarshal(data, &next); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	for name, r := range next {
		if r.Percent < 0 || r.Percent > 100 {
			return fmt.Errorf("flag %q: percent %d out of range", name, r.Percent)
		}
		if r.Expires != "" {
			if _, err := time.Parse(time.DateOnly, r.Expires); err != nil {
				return fmt.Errorf("flag %q: bad expires: %w", name, err)
			}
		}
	}
	s.cur.Store(&next)
	return nil
}

// Snapshot 回傳目前生效的 Set；尚未載入時回傳空的 Set。
func (s *Store) Snapshot() Set {
	if p := s.cur.Load(); p != nil {
		return *p
	}
	return Set{}
}

// Enabled 回答「這位使用者現在該不該走新行為」。
// 未知的旗標一律關閉：查不到設定時，走舊行為最安全。
func (s *Store) Enabled(name, userID string) bool {
	r, ok := s.Snapshot()[name]
	if !ok || !r.Enabled {
		return false
	}
	if slices.Contains(r.AllowUsers, userID) {
		return true
	}
	return bucket(name, userID) < r.Percent
}

// bucket 把 (旗標, 使用者) 穩定地對應到 0–99。
// 加上旗標名稱當鹽，避免同一批使用者在每個旗標都是「最先被放量」的那群。
func bucket(name, userID string) int {
	h := fnv.New32a()
	h.Write([]byte(name + ":" + userID))
	return int(h.Sum32() % 100)
}
