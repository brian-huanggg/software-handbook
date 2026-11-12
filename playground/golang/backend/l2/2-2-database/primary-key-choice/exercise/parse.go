package exercise

import (
	"errors"

	pkchoice "handbook/playground/golang/backend/primary-key-choice"
)

// ParseV7 把 "0190a5f2-3c1e-7abc-8def-0123456789ab" 這種字串轉回 UUID。
// 必須拒絕：長度不對、不是十六進位、版本不是 7、variant 不是 10xx。
func ParseV7(s string) (pkchoice.UUID, error) {
	return pkchoice.UUID{}, errors.New("not implemented")
}
