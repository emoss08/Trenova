package proposalexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRuns struct {
	run  *agent.AgentRun
	err  error
	asks []agentports.GetAgentRunByIDRequest
}

func (r *stubRuns) GetByID(
	_ context.Context,
	req agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	r.asks = append(r.asks, req)

	return r.run, r.err
}

type conversationTool struct {
	recordingTool

	validatedFor pulid.ID
}

func (t *conversationTool) Validate(_ context.Context, params agentports.ToolExecuteParams) error {
	t.validatedFor = params.ThreadID

	return nil
}

func newConversationTool() *conversationTool {
	return &conversationTool{recordingTool: recordingTool{
		name:      "edi_apply_connection_draft",
		resource:  permission.ResourceEDI,
		operation: permission.OpUpdate,
	}}
}

func TestExecute_HandsTheToolTheConversationItsRunAnswered(t *testing.T) {
	t.Parallel()

	tool := newConversationTool()
	proposal := testProposal(tool.name, map[string]any{})
	proposal.RunID = pulid.MustNew("arun_")
	thread := pulid.MustNew("athr_")
	runs := &stubRuns{run: &agent.AgentRun{
		ID:          proposal.RunID,
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   thread,
	}}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.runs = runs

	require.NoError(t, executor.Execute(
		t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	))

	assert.Equal(t, thread, tool.validatedFor, "the check before the write reads the conversation")
	assert.Equal(t, thread, tool.lastParams.ThreadID, "the write reads the conversation")
	require.Len(t, runs.asks, 1, "the run is read once for the approval")
	assert.Equal(t, proposal.RunID, runs.asks[0].ID)
	require.NotNil(t, runs.asks[0].TenantInfo)
	assert.Equal(t, proposal.OrganizationID, runs.asks[0].TenantInfo.OrgID)
	assert.Equal(t, proposal.BusinessUnitID, runs.asks[0].TenantInfo.BuID)
}

func TestExecute_HandsNoConversationForARunAboutSomethingElse(t *testing.T) {
	t.Parallel()

	tool := newConversationTool()
	proposal := testProposal(tool.name, map[string]any{})
	proposal.RunID = pulid.MustNew("arun_")
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.runs = &stubRuns{run: &agent.AgentRun{
		ID:          proposal.RunID,
		SubjectType: agent.SubjectShipment,
		SubjectID:   pulid.MustNew("shp_"),
	}}

	require.NoError(t, executor.Execute(
		t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	))

	assert.True(t, tool.lastParams.ThreadID.IsNil(),
		"a shipment's id is never handed over as a conversation")
}

func TestExecute_RefusesWhenTheRunCannotBeRead(t *testing.T) {
	t.Parallel()

	tool := newConversationTool()
	proposal := testProposal(tool.name, map[string]any{})
	proposal.RunID = pulid.MustNew("arun_")
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	executor.runs = &stubRuns{err: errors.New("database is down")}

	err := executor.Execute(
		t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Zero(t, tool.calls, "a write that needs its conversation never runs without it")
}

func TestExecute_ReadsNoRunForAProposalWithoutOne(t *testing.T) {
	t.Parallel()

	tool := newConversationTool()
	proposal := testProposal(tool.name, map[string]any{})
	runs := &stubRuns{err: errors.New("no run")}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.runs = runs

	require.NoError(t, executor.Execute(
		t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	))

	assert.Empty(t, runs.asks)
	assert.True(t, tool.lastParams.ThreadID.IsNil())
}
