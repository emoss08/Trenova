package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
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
					Name:      "report_update",
					Arguments: map[string]any{"definitionId": "rd_1"},
				},
				{
					ID:        "c2",
					Name:      "report_update",
					Arguments: map[string]any{"definitionId": "rd_2"},
				},
				{ID: "c3", Name: "report_get_run", Arguments: map[string]any{"runId": "rrun_1"}},
			},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_update",
			Content:    "Recorded a proposal to run \"report_update\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c2",
			ToolName:   "report_update",
			Content:    "Recorded a proposal to run \"report_update\". It is awaiting a person's review.",
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c3",
			ToolName:   "report_get_run",
			Content:    "{\"status\":\"succeeded\"}",
		},
		{Role: conversation.RoleAssistant, Content: "Proposed two updates."},
	}
	outcomes := []agentruntimeports.ProposalOutcome{
		{
			SourceMessageID: sourceID,
			ToolName:        "report_update",
			Status:          agent.ProposalStatusExecutionFailed,
			ExecutionError:  "validation failed: unknown field \"\" on entity \"shipment_move\"",
		},
		{
			SourceMessageID: sourceID,
			ToolName:        "report_update",
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
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "report_fork"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_fork",
			Content:    "Recorded a proposal",
		},
	}
	outcomes := []agentruntimeports.ProposalOutcome{
		{SourceMessageID: sourceID, ToolName: "report_fork", Status: agent.ProposalStatusPending},
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
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "report_fork"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "report_fork",
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

	action := actionTool("report_update", agent.TierActWithApproval, nil)
	args := map[string]any{"definitionId": "rd_1", "name": "Readable"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("report_update", args),
		textTurn("It is waiting on your card."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := testDefinition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
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
		toolTurn("report_update", args),
		textTurn("Proposed again."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := testDefinition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
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
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := testDefinition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "update",
	})
	require.NoError(t, err)
	assert.Len(t, result.Actions, 1)
}

/*
Arguments that cannot be encoded are no one's duplicate.

Both sets encoded to the same empty key, so a second proposal whose arguments
also failed to encode was dropped as a copy of the first, whatever it asked
for.
*/
func TestPendingDuplicate_NeverMatchesArgumentsThatCannotBeEncoded(t *testing.T) {
	t.Parallel()

	call := llmports.ToolCall{
		ID:        "call_2",
		Name:      "report_update",
		Arguments: map[string]any{"definitionId": "rd_2", "cycle": make(chan int)},
	}
	earlier := []agentruntimeports.ProposalOutcome{{
		ToolName:   "report_update",
		Status:     agent.ProposalStatusPending,
		ToolParams: map[string]any{"definitionId": "rd_1", "cycle": make(chan int)},
	}}
	thisTurn := []agentruntimeports.PendingAction{{
		ToolName:  "report_update",
		Arguments: map[string]any{"definitionId": "rd_1", "cycle": make(chan int)},
	}}

	require.Empty(t, argumentsKey(call.Arguments), "the arguments must fail to encode")
	assert.False(t, pendingDuplicate(call, nil, thisTurn),
		"a different proposal earlier in this turn is not this one")
	assert.False(t, pendingDuplicate(call, earlier, nil),
		"a different proposal from an earlier turn is not this one")
	assert.False(t, pendingDuplicate(call, earlier, thisTurn))
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
		toolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		textTurn("I will fix the column."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	definition := testDefinition("report_update")
	definition.AutonomyCeiling = agent.TierActWithApproval
	actor := testActor()

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
		toolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		textTurn("Proposed."),
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

// A change the approver made before approving is part of what ran, and the
// model that proposed the original values is told, so its reply describes
// the message that was sent rather than the one it drafted.
func TestProposalOutcomeText_SaysWhatTheApproverChanged(t *testing.T) {
	t.Parallel()

	text := proposalOutcomeText("worker_send_message", agentruntimeports.ProposalOutcome{
		Status:        agent.ProposalStatusExecuted,
		Modifications: map[string]any{"priority": "high", "message": "Call dispatch now"},
	})

	assert.Contains(t, text, `after changing message = "Call dispatch now", priority = "high"`)
	assert.Contains(t, text, "ran successfully")

	plain := proposalOutcomeText(
		"worker_send_message",
		agentruntimeports.ProposalOutcome{Status: agent.ProposalStatusExecuted},
	)
	assert.NotContains(t, plain, "after changing")
}

// A proposal's result is replayed as what became of it. The rate its preview
// priced is still something the model read, and a reply quoting it a turn
// later was rewritten as if it had made the figure up.
func TestReplayHistory_KeepsTheFiguresOfAProposalsResultAsRead(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("amsg_")
	filed := "Summary: The rate agreement would charge 5959.35 USD for the linehaul."
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Book it"},
		{
			ID:        sourceID,
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "shipment_create"}},
		},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   "shipment_create",
			Content:    filed,
		},
	}
	outcomes := []agentruntimeports.ProposalOutcome{{
		SourceMessageID: sourceID,
		ToolName:        "shipment_create",
		Status:          agent.ProposalStatusPending,
	}}

	messages, shortened := replayHistory(history, outcomes)

	require.Len(t, messages, 3)
	assert.NotContains(t, messages[2].Content, "5959.35")
	assert.Equal(t, []string{filed}, shortened)
}
