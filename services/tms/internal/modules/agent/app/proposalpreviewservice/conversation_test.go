package proposalpreviewservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRuns struct {
	run *agent.AgentRun
	err error
}

func (r stubRuns) GetByID(
	context.Context,
	agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return r.run, r.err
}

func TestForProposal_PreviewsInTheConversationTheRunAnswered(t *testing.T) {
	t.Parallel()

	f := newFixture()
	thread := pulid.MustNew("athr_")
	f.svc.runs = stubRuns{run: &agent.AgentRun{
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   thread,
	}}

	_, err := f.svc.ForProposal(t.Context(), &agentruntimeports.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	require.NotEmpty(t, f.tool.asked)
	assert.Equal(t, thread, f.tool.asked[0].ThreadID,
		"the preview reads what was attached to the conversation, as the write will")
}

func TestForProposal_SaysWhenTheRunCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.svc.runs = stubRuns{err: errors.New("database is down")}

	_, err := f.svc.ForProposal(t.Context(), &agentruntimeports.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})

	require.Error(t, err)
	assert.Empty(t, f.tool.asked)
}
