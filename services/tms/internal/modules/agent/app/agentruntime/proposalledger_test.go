package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The person said "yes" to a card they had not clicked, and the model raised
// the same write again. An identical pending proposal is refused as a
// duplicate, with the model told to point at the card.
func TestRun_DoesNotProposeWhatIsAlreadyPending(t *testing.T) {
	t.Parallel()

	action := actionTool("report_update", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1", "name": "Readable"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_update", args),
		runtimetest.TextTurn("It is waiting on your card."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := runtimetest.Definition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "yes",
		Proposals: []agentruntimeports.ProposalOutcome{{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "report_update",
			ToolParams:      map[string]any{"name": "Readable", "definitionId": "rd_1"},
			Status:          agent.ProposalStatusPending,
		}},
	})
	require.NoError(t, err)

	assert.Empty(t, result.Actions, "no second card")
	assert.Contains(t, result.Messages[2].Content, "already waiting")
	assert.False(t, result.Messages[2].ToolFailed)
}

func TestRun_StillProposesWhenTheEarlierOneWasDecided(t *testing.T) {
	t.Parallel()

	action := actionTool("report_update", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_update", args),
		runtimetest.TextTurn("Proposed again."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := runtimetest.Definition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "try again",
		Proposals: []agentruntimeports.ProposalOutcome{{
			ToolName:   "report_update",
			ToolParams: args,
			Status:     agent.ProposalStatusRejected,
		}},
	})
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

func TestRun_DoesNotProposeTheSameWriteTwiceInOneTurn(t *testing.T) {
	t.Parallel()

	action := actionTool("report_update", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		{
			ToolCalls: []llmports.ToolCall{
				{ID: "call_1", Name: "report_update", Arguments: args},
				{
					ID:        "call_2",
					Name:      "report_update",
					Arguments: map[string]any{"definitionId": "rd_1"},
				},
			},
			ModelIdentifier: "test-model",
		},
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := runtimetest.Definition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "update",
	})
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

type validatingActionTool struct {
	*stubActionToolAlias

	invalid  error
	checked  int
	lastSeen agenttoolports.ToolExecuteParams
}

type stubActionToolAlias = agenttest.StubActionTool

func (t *validatingActionTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	t.checked++
	t.lastSeen = params

	return t.invalid
}

// A tool that can check its arguments is asked before the proposal exists,
// so a definition that would fail on execution is refused to the model now
// rather than after a person approved it.
func TestRun_RefusesAProposalItsToolSaysWouldFail(t *testing.T) {
	t.Parallel()

	tool := &validatingActionTool{
		stubActionToolAlias: actionTool("report_update", agent.TierActWithApproval, nil),
		invalid: errors.New(
			"unknown field \"assignment.primaryWorker.firstName\" on entity \"shipment_move\"",
		),
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		runtimetest.TextTurn("I will fix the column."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	definition := runtimetest.Definition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval
	actor := runtimetest.Actor()

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      actor,
		Input:      "update",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.checked)
	assert.Equal(t, actor.OrganizationID, tool.lastSeen.OrganizationID)
	assert.Empty(t, result.Actions)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "would fail as called")
	assert.Contains(t, result.Messages[2].Content, "unknown field")

	tool.invalid = nil
	completion = &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		runtimetest.TextTurn("Proposed."),
	}}
	rt = newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	result, err = rt.Run(
		t.Context(),
		&agentruntimeports.RunRequest{Definition: definition, Actor: actor, Input: "update"},
	)
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}
