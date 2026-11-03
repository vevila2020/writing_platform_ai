package session_test

import (
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/session"
	"github.com/vevila2020/writing_platform_ai/agent/session/sessiontest"
)

func TestMemoryStoreContract(t *testing.T) {
	sessiontest.RunStoreContract(t, func(testing.TB) session.Store {
		return session.Memory()
	})
}
