package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type threadOnlySpecReader struct {
	fakeSpecReader
}

func (r *threadOnlySpecReader) Read(
	ctx context.Context,
	req *edispec.ReadRequest,
) (*ediports.SpecDocument, error) {
	if req.ThreadID.IsNil() {
		return nil, edispec.ErrNoConversation
	}
	return r.fakeSpecReader.Read(ctx, req)
}

type threadRun struct {
	run *agent.AgentRun
}

func (r threadRun) GetByID(
	context.Context,
	agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return r.run, nil
}

func TestApplyConnectionDraft_RunsFromTheParametersAnApprovalHandsIt(t *testing.T) {
	t.Parallel()

	drafts := &fakeConnectionDrafts{plan: readyPlan(), guard: &agenttoolstest.WriteGuard{}}
	reader := &threadOnlySpecReader{}
	tool := NewApplyConnectionDraftTool(drafts, reader).(*ApplyConnectionDraftTool)
	thread := pulid.MustNew("ath_")
	proposal := &agent.AgentProposal{
		ID:             pulid.MustNew("agp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		RunID:          pulid.MustNew("arun_"),
		ToolName:       tool.Name(),
		ToolParams:     draftArgs(pulid.MustNew("doc_")),
	}
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
	}
	policy := tool.Policy()
	bare := proposalexecutor.ExecutionParams(proposal, &policy, proposal.ToolParams, actor)

	require.ErrorIs(t, tool.Validate(t.Context(), bare), edispec.ErrNoConversation,
		"without the conversation the guide cannot be read")

	params, err := proposalexecutor.WithConversation(t.Context(), threadRun{run: &agent.AgentRun{
		ID:          proposal.RunID,
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   thread,
	}}, bare)
	require.NoError(t, err)

	require.NoError(t, tool.Validate(t.Context(), params))
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, preview.Warnings)
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, reader.read)
	assert.Equal(t, thread, reader.read.ThreadID)
	require.NotNil(t, drafts.applied)
	assert.Equal(t, draftDigest, drafts.applied.ExpectedDocumentDigest)
	assert.Equal(t, actor, drafts.applied.Actor)
}

func TestApplyConnectionDraft_ProposesItsArgumentsWithSecretsRedacted(t *testing.T) {
	t.Parallel()

	tool := NewApplyConnectionDraftTool(&fakeConnectionDrafts{plan: readyPlan()}, &fakeSpecReader{})
	resolver, resolves := tool.(agentports.ToolSelectionResolver)
	require.True(t, resolves, "the runtime settles the arguments before the proposal is stored")

	args := draftArgs(pulid.MustNew("doc_"))
	args[edispec.ParamValues] = []any{map[string]any{
		"field": "transport.username", "value": "acme", "confidence": 0.95,
		"citation": map[string]any{"page": 2, "quote": "Username: acme Password: Acme#Test2026"},
	}}

	resolved, err := resolver.ResolveSelection(t.Context(), agenttoolstest.AgentParamsFor(args))
	require.NoError(t, err)

	input, err := edispec.Parse(resolved, pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, input.Draft.Values, 1)
	assert.Equal(t, "Username: acme Password: [redacted]", input.Draft.Values[0].Citation.Quote)
	assert.Equal(t, "acme", input.Draft.Values[0].Value)
	assert.Equal(t, draftDigest, input.Digest)
}
