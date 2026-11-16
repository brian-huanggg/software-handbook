// Package shortener 是短網址服務 v1 的一小塊：建立短碼，以及判斷目標網址能不能接受。
package shortener

import (
	"errors"
	"strconv"
)

// ErrCodeTaken 表示短碼已經被用走。
var ErrCodeTaken = errors.New("code taken")

// Store 是記憶體裡的短碼表，用來示範路徑覆蓋；真正的 v1 存在 SQL。
type Store struct {
	links map[string]string
	next  int
}

// NewStore 建立空的短碼表。
func NewStore() *Store {
	return &Store{links: map[string]string{}}
}

func (s *Store) has(code string) bool {
	_, ok := s.links[code]
	return ok
}

// nextCode 用遞增的數字當短碼，讓測試可以預測下一個自動短碼是什麼。
func (s *Store) nextCode() string {
	s.next++
	return strconv.Itoa(s.next)
}

// CreateNaive 是「先想想」裡的版本：兩個前後接續的判斷，分支覆蓋兩個案例就滿，
// 但「自動產生的短碼剛好被自訂短碼佔走」這條路徑沒人走過。
func (s *Store) CreateNaive(target, alias string) (string, error) {
	code := alias
	if code == "" {
		code = s.nextCode()
	}
	if s.has(code) {
		return "", ErrCodeTaken
	}
	s.links[code] = target
	return code, nil
}

// Create 是修正後的版本：只有使用者自己指定的短碼衝突才回錯誤，
// 自動產生的短碼衝突就換下一個。
func (s *Store) Create(target, alias string) (string, error) {
	if alias != "" {
		if s.has(alias) {
			return "", ErrCodeTaken
		}
		s.links[alias] = target
		return alias, nil
	}
	code := s.nextCode()
	for s.has(code) {
		code = s.nextCode()
	}
	s.links[code] = target
	return code, nil
}
