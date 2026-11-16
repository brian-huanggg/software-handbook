package shortener

import "errors"

// ErrRejected 表示目標網址不被接受。
var ErrRejected = errors.New("target rejected")

// ValidateTarget 只接受 http 與 https，而且 host 不能是空的。
func ValidateTarget(scheme, host string) error {
	if !(scheme == "http" || scheme == "https") || host == "" {
		return ErrRejected
	}
	return nil
}
