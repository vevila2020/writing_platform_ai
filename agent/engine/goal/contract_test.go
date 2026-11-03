package goal_test

import (
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/engine/goal"
	"github.com/vevila2020/writing_platform_ai/agent/engine/goal/goaltest"
)

func TestStandardManagerContract(t *testing.T) {
	goaltest.RunStandardManagerContract(t, func(testing.TB) goal.GoalManager {
		return goal.Standard()
	})
}
