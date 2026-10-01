package agentjobs

import (
	"context"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type filedProposals struct {
	repositories.AgentProposalRepository
}

func (filedProposals) ListByRun(
	_ context.Context,
	req repositories.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return []*agent.AgentProposal{{
		ID:     pulid.MustNew("apr_"),
		RunID:  req.RunID,
		Status: agent.ProposalStatusExecuted,
	}}, nil
}

type settledRuns struct {
	repositories.AgentRunRepository

	updated []*agent.AgentRun
}

func (r *settledRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{ID: req.ID, Status: agent.RunStatusDiagnosing}, nil
}

func (r *settledRuns) Update(_ context.Context, run *agent.AgentRun) (*agent.AgentRun, error) {
	r.updated = append(r.updated, run)

	return run, nil
}

func settle(t *testing.T, messages []conversation.Message) *agent.AgentRun {
	t.Helper()

	runs := &settledRuns{}
	a := &Activities{logger: zap.NewNop(), runRepo: runs, proposalRepo: filedProposals{}}
	_, run, err := a.settleRun(t.Context(), settleRunParams{
		Payload: &AgentRunPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			},
			RunID: pulid.MustNew("arun_"),
		},
		Outcome: &serviceports.RunResult{Reply: "Filed.", Messages: messages},
	})
	require.NoError(t, err)
	require.Len(t, runs.updated, 1)
	assert.Same(t, run, runs.updated[0])

	return run
}

func TestSettleRun_KeepsTheRunsTranscript(t *testing.T) {
	t.Parallel()

	run := settle(t, []conversation.Message{
		{
			Role:      conversation.RoleAssistant,
			Content:   "Checking the shipment.",
			ToolCalls: []conversation.ToolCallRecord{{ID: "c1", Name: "update_worker"}},
			CreatedAt: 10,
		},
		{
			Role:        conversation.RoleTool,
			ToolCallID:  "c1",
			ToolName:    "update_worker",
			ToolFailed:  true,
			ToolVerdict: "denied",
			Content:     `Tool "update_worker" is not permitted.`,
			CreatedAt:   11,
		},
		{Role: conversation.RoleAssistant, Content: "Filed.", CreatedAt: 12},
	})

	assert.Equal(t, "Filed.", run.Summary)
	require.NotNil(t, run.Transcript)
	require.Len(t, run.Transcript.Messages, 3)
	assert.Zero(t, run.Transcript.OmittedMessages)
	assert.Equal(t, "update_worker", run.Transcript.Messages[0].ToolCalls[0].Name)
	assert.Equal(t, "denied", run.Transcript.Messages[1].ToolVerdict)
	assert.True(t, run.Transcript.Messages[1].ToolFailed)
	assert.Equal(t, "Filed.", run.Transcript.Messages[2].Content)
}

func TestSettleRun_BoundsALongRunsTranscript(t *testing.T) {
	t.Parallel()

	chunk := strings.Repeat("x", 20*1024)
	messages := make([]conversation.Message, 0, 40)
	for idx := range 40 {
		messages = append(messages, conversation.Message{
			Role:      conversation.RoleTool,
			ToolName:  "get_shipment",
			Content:   chunk,
			CreatedAt: int64(idx),
		})
	}
	messages = append(messages, conversation.Message{
		Role:      conversation.RoleTool,
		ToolName:  "run_report",
		Content:   strings.Repeat("y", agent.TranscriptMessageBytes),
		CreatedAt: 40,
	})

	run := settle(t, messages)

	require.NotNil(t, run.Transcript)
	encoded, err := sonic.Marshal(run.Transcript)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), agent.TranscriptBytes+1024)

	kept := run.Transcript.Messages
	assert.Positive(t, run.Transcript.OmittedMessages)
	assert.Equal(t, len(messages), len(kept)+run.Transcript.OmittedMessages)
	assert.Equal(t, int64(0), kept[0].CreatedAt, "the run's opening is kept")
	last := kept[len(kept)-1]
	assert.Equal(t, int64(40), last.CreatedAt, "and so is its end")
	assert.True(t, last.Omitted, "a message over the bound keeps only who said it")
	assert.Empty(t, last.Content)
	assert.Equal(t, "run_report", last.ToolName)
	assert.Equal(t, kept[run.Transcript.OmittedAt-1].CreatedAt+int64(run.Transcript.OmittedMessages)+1,
		kept[run.Transcript.OmittedAt].CreatedAt, "the gap is counted where it fell")
}

func TestSettleRun_LeavesNoTranscriptForARunThatSaidNothing(t *testing.T) {
	t.Parallel()

	run := settle(t, nil)

	assert.Nil(t, run.Transcript)
}
