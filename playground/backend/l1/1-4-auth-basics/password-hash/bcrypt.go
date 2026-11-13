package passwordhash

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashBcrypt 的輸出已包含 cost 與 salt，不需要另外存。
// 密碼超過 72 bytes 時 bcrypt 會回傳 bcrypt.ErrPasswordTooLong，
// 這裡直接把錯誤交給呼叫端，而不是默默截斷。
func HashBcrypt(password string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyBcrypt(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return err == nil, err
}
