package token_test

import (
	"testing"

	"github.com/google/uuid"

	"authkit/token"
	"authkit/token/storetest"
)

func TestMemoryRefreshStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) (token.RefreshStore, func() string) {
		return token.NewMemoryRefreshStore(), uuid.NewString
	})
}
