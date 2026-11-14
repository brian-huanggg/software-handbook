package session_test

import (
	"testing"

	"github.com/google/uuid"

	"authkit/session"
	"authkit/session/storetest"
)

func TestMemoryStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) (session.Store, func() string) {
		return session.NewMemoryStore(), uuid.NewString
	})
}
