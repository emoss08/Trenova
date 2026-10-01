package agentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRunParentOwnerKind_IsAbsentForARunNothingHandedATask(t *testing.T) {
	t.Parallel()

	kind, err := (&AgentRunResolver{}).ParentOwnerKind(t.Context(), &agent.AgentRun{})
	require.NoError(t, err)
	assert.Nil(t, kind)

	kind, err = (&AgentRunResolver{}).ParentOwnerKind(t.Context(), &agent.AgentRun{ParentOwnerKind: agent.RunOwnerAssistantTurn})
	require.NoError(t, err)
	require.NotNil(t, kind)
	assert.Equal(t, gqlmodel.AgentRunEventOwnerKindAssistantTurn, *kind)
}
