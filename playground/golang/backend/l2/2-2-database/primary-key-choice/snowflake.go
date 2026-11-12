package pkchoice

import (
	"errors"
	"sync"
	"time"
)

// Snowflake 的 64 bits 配置：1 bit 保留（維持正數）、41 bits 毫秒時間、
// 10 bits 機器編號、12 bits 同一毫秒內的序號。
const (
	workerBits   = 10
	sequenceBits = 12
	maxWorkerID  = 1<<workerBits - 1
	maxSequence  = 1<<sequenceBits - 1
)

// Epoch 是自訂紀元。41 bits 毫秒大約能用 69 年，起點越晚越耐用。
var Epoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

var ErrClockBackwards = errors.New("clock moved backwards")

type Snowflake struct {
	mu       sync.Mutex
	workerID int64
	lastMS   int64
	sequence int64
	now      func() time.Time
}

func NewSnowflake(workerID int64, now func() time.Time) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, errors.New("worker id out of range")
	}
	if now == nil {
		now = time.Now
	}
	return &Snowflake{workerID: workerID, lastMS: -1, now: now}, nil
}

// Next 回傳下一個 ID。時鐘倒退時直接回錯誤，不偷偷重用舊的毫秒，
// 否則兩台機器的時鐘校正後會產生重複 ID。
func (s *Snowflake) Next() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ms := s.now().Sub(Epoch).Milliseconds()
	switch {
	case ms < s.lastMS:
		return 0, ErrClockBackwards
	case ms == s.lastMS:
		s.sequence++
		if s.sequence > maxSequence {
			// 這一毫秒的 4096 個序號用完，等到下一毫秒。
			for ms <= s.lastMS {
				ms = s.now().Sub(Epoch).Milliseconds()
			}
			s.sequence = 0
		}
	default:
		s.sequence = 0
	}
	s.lastMS = ms

	return ms<<(workerBits+sequenceBits) | s.workerID<<sequenceBits | s.sequence, nil
}
