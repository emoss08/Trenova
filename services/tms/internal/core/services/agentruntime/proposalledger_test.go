package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A proposal's tool result said "awaiting review" forever, whatever the
// person had decided since, so the model re-proposed and misreported. The
// replay now carries each proposal's current state.
func TestToAdapterMessages_ReplaysWhatBecameOfEachProposal(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	executedAt := int64(1_790_000_000)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Fix the dates"},
		{
			ID:   sourceID,
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{
					ID:        "c1",
					Name:      "update_report",
					Arguments: map[string]any{"definitionId": "rd_1"},
				},
				{
					ID:        "c2",
					Name:      "update_report",
					Arguments: map[string]any{"definitionId": "rd_2"},
				},
				{ID: "c3", Name: "get_report_run", Arguments: map[string]any{"runId": "rrun_1"}},
			},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "update_report",
			Content:    "Recorded a proposal to run \"update_report\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c2",
			ToolName:   "update_report",
			Content:    "Recorded a proposal to run \"update_report\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c3",
			ToolName:   "get_report_run",
			Content:    "{\"status\":\"succeeded\"}",
		},
		{Role: conversation.RoleAssistant, Content: "Proposed two updates."},
	}
	outcomes := []serviceports.ProposalOutcome{
		{
			SourceMessageID: sourceID,
			ToolName:        "update_report",
			Status:          agent.ProposalStatusExecutionFailed,
			ExecutionError:  "validation failed: unknown field \"\" on entity \"shipment_move\"",
		},
		{
			SourceMessageID: sourceID,
			ToolName:        "update_report",
			Status:          agent.ProposalStatusExecuted,
			ExecutedAt:      &executedAt,
		},
	}

	messages := toAdapterMessages(history, outcomes)
	require.Len(t, messages, 6)

	assert.Contains(t, messages[2].Content, "FAILED")
	assert.Contains(t, messages[2].Content, "unknown field")
	assert.Contains(t, messages[3].Content, "ran successfully")
	assert.Equal(
		t,
		"{\"status\":\"succeeded\"}",
		messages[4].Content,
		"a plain tool result is left alone",
	)
}

func TestToAdapterMessages_TellsTheModelAPendingProposalIsStillWaiting(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Copy it"},
		{
			ID:        sourceID,
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "fork_report"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "fork_report",
			Content:    "Recorded a proposal",
		},
	}
	outcomes := []serviceports.ProposalOutcome{
		{SourceMessageID: sourceID, ToolName: "fork_report", Status: agent.ProposalStatusPending},
	}

	messages := toAdapterMessages(history, outcomes)
	require.Len(t, messages, 3)
	assert.Contains(t, messages[2].Content, "still waiting")
	assert.Contains(t, messages[2].Content, "Do not propose it again")
}

func TestToAdapterMessages_LeavesResultsAloneWithNoOutcomes(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Copy it"},
		{
			ID:        pulid.MustNew("amsg_"),
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "fork_report"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "fork_report",
			Content:    "Recorded a proposal",
		},
	}

	messages := toAdapterMessages(history, nil)
	assert.Equal(t, "Recorded a proposal", messages[2].Content)
}

// The person said "yes" to a card they had not clicked, and the model raised
// the same write again. An identical pending proposal is refused as a
// duplicate, with the model told to point at the card.
func TestRun_DoesNotProposeWhatIsAlreadyPending(t *testing.T) {
	t.Parallel()

	action := actionTool("update_report", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1", "name": "Readable"}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", args),
		textTurn("It is waiting on your card."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{action}}, nil)
	definition := testDefinition("update_report")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "yes",
		Proposals: []serviceports.ProposalOutcome{{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "update_report",
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

	action := actionTool("update_report", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1"}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", args),
		textTurn("Proposed again."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{action}}, nil)
	definition := testDefinition("update_report")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "try again",
		Proposals: []serviceports.ProposalOutcome{{
			ToolName:   "update_report",
			ToolParams: args,
			Status:     agent.ProposalStatusRejected,
		}},
	})
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

func TestRun_DoesNotProposeTheSameWriteTwiceInOneTurn(t *testing.T) {
	t.Parallel()

	action := actionTool("update_report", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1"}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		{
			ToolCalls: []serviceports.ToolCall{
				{ID: "call_1", Name: "update_report", Arguments: args},
				{
					ID:        "call_2",
					Name:      "update_report",
					Arguments: map[string]any{"definitionId": "rd_1"},
				},
			},
			ModelIdentifier: "test-model",
		},
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{action}}, nil)
	definition := testDefinition("update_report")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "update",
	})
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

type validatingActionTool struct {
	*stubActionToolAlias

	invalid  error
	checked  int
	lastSeen serviceports.ToolExecuteParams
}

type stubActionToolAlias = agentruntimetest.StubActionTool

func (t *validatingActionTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
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
		stubActionToolAlias: actionTool("update_report", agent.TierActWithApproval, nil),
		invalid: errors.New(
			"unknown field \"assignment.primaryWorker.firstName\" on entity \"shipment_move\"",
		),
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", map[string]any{"definitionId": "rd_1"}),
		textTurn("I will fix the column."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	definition := testDefinition("update_report")
	definition.AutonomyCeiling = agent.TierActWithApproval
	actor := testActor()

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
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
	completion = &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", map[string]any{"definitionId": "rd_1"}),
		textTurn("Proposed."),
	}}
	rt = newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	result, err = rt.Run(
		t.Context(),
		&serviceports.RunRequest{Definition: definition, Actor: actor, Input: "update"},
	)
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

// A change the approver made before approving is part of what ran, and the
// model that proposed the original values is told, so its reply describes
// the message that was sent rather than the one it drafted.
func TestProposalOutcomeText_SaysWhatTheApproverChanged(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("notify_driver", serviceports.ProposalOutcome{
		Status:        agent.ProposalStatusExecuted,
		Modifications: map[string]any{"priority": "high", "message": "Call dispatch now"},
	})

	assert.Contains(t, text, `after changing message = "Call dispatch now", priority = "high"`)
	assert.Contains(t, text, "ran successfully")

	plain := proposalOutcomeText(
		"notify_driver",
		serviceports.ProposalOutcome{Status: agent.ProposalStatusExecuted},
	)
	assert.NotContains(t, plain, "after changing")
}
