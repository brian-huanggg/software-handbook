package session_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/brian-huanggg/software-handbook/templates/authkit/session"
	"github.com/brian-huanggg/software-handbook/templates/authkit/session/storetest"
)

func TestMemoryStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) (session.Store, func() string) {
		return session.NewMemoryStore(), uuid.NewString
	})
}
