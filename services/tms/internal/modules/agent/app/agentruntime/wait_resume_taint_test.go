package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenTurn_AResumedWaitLeavesTheNextTurnCappedAndTainted(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	request := func(taint *agent.RunTaint, outside bool) *agentruntimeports.RunRequest {
		return &agentruntimeports.RunRequest{
			Definition:           runtimetest.Definition(),
			Actor:                runtimetest.Actor(),
			Input:                "Carry on with what the portal said.",
			Unattended:           true,
			Context:              agentturn.RuntimeContext{Trigger: agent.RunTriggerEvent},
			Taint:                taint,
			AfterExternalContent: outside,
		}
	}

	resumed := rt.OpenTurn(t.Context(), request(nil, true))
	thread := &conversation.Thread{}
	require.True(t, thread.AbsorbTaint(resumed.Taint(), 10))

	next := rt.OpenTurn(t.Context(), request(thread.Taint, false))

	assert.True(t, next.Taint().Tainted())
	assert.True(t, next.ReadExternalContent(), "every write of the next turn is proposed")
}
