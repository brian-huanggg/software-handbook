package token_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/brian-huanggg/software-handbook/templates/authkit/token"
	"github.com/brian-huanggg/software-handbook/templates/authkit/token/storetest"
)

func TestMemoryRefreshStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) (token.RefreshStore, func() string) {
		return token.NewMemoryRefreshStore(), uuid.NewString
	})
}
