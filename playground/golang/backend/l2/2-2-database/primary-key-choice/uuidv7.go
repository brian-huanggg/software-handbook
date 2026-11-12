package pkchoice

import (
	"crypto/rand"
	"fmt"
	"time"
)

// UUID 是 16 bytes 的識別碼，直接對應資料庫的 uuid／BINARY(16) 欄位。
type UUID [16]byte

// NewV7 依 RFC 9562 產生 UUID v7：前 48 bits 是 Unix 毫秒時間戳，
// 其餘是版本、variant 與隨機位元。字串或 bytes 比大小就等於比時間先後。
func NewV7(now time.Time) (UUID, error) {
	var u UUID
	if _, err := rand.Read(u[:]); err != nil {
		return UUID{}, fmt.Errorf("read random: %w", err)
	}

	ms := uint64(now.UnixMilli())
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)

	u[6] = (u[6] & 0x0f) | 0x70 // version 7
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10xx
	return u, nil
}

// Time 取回內嵌的時間戳。這也是 v7 的隱私代價：任何人拿到 ID 就知道它何時建立。
func (u UUID) Time() time.Time {
	ms := uint64(u[0])<<40 | uint64(u[1])<<32 | uint64(u[2])<<24 |
		uint64(u[3])<<16 | uint64(u[4])<<8 | uint64(u[5])
	return time.UnixMilli(int64(ms))
}

func (u UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
