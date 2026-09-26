package proposalexecutor

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A tool that checks its own arguments is asked again where the write runs,
// with or without an approver's changes: the world may have moved since the
// proposal was filed, and a write that would fail is refused before an
// execution is recorded.
func TestExecute_RefusesAWriteItsToolSaysWouldFail(t *testing.T) {
	t.Parallel()

	refusal := errors.New("that driver has no portal access")
	tool := &schemaTool{validateErr: refusal}
	repo := &fakeProposalRepo{}
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	err := newExecutor(tool, repo, &fakePermissions{allowed: true}).
		Execute(t.Context(), proposal, nil, actor)

	require.ErrorIs(t, err, refusal)
	assert.Nil(t, tool.ran, "the tool never ran")
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
	assert.Contains(t, repo.recorded[0].errText, "portal access")
}

func TestExecute_RunsAWriteItsToolAdmits(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	require.NoError(t, newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true}).
		Execute(t.Context(), proposal, nil, actor))

	assert.Equal(t, map[string]any{"workerId": "wrk_1", "message": "Call in"}, tool.ran)
}
