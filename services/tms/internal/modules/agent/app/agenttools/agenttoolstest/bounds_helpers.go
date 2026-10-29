package agenttoolstest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/require"
)

func BoundsOf(t *testing.T, tool agenttoolports.AgentTool) func(
	context.Context,
	*agenttoolports.ToolExecuteParams,
) agenttoolports.MoneyBound {
	t.Helper()

	policy, ok := agenttoolports.PolicyOf(tool)
	require.True(t, ok)
	require.True(t, policy.Bounded(), "%s proves its bounds", policy.Name)

	return policy.Condition.Bounds
}
