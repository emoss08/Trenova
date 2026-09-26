package proposalexecutor

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type selfScopedTool struct {
	*recordingTool
}

func (t selfScopedTool) Policy() services.ToolPolicy {
	return selfScopedPolicy(t.recordingTool.Policy())
}

func selfScopedPolicy(policy services.ToolPolicy) services.ToolPolicy {
	policy.Scope = agent.ToolScopeSelf
	policy.Egress = []agent.EgressClass{agent.EgressPersonal}
	policy.MaxTier = agent.TierAutoExecute

	return policy
}

/*
A change to the person's own home page is approved without a role grant, as
the home page's editor lets them make it — but only by a person. The tool
itself checks the approver is the person it was proposed for.
*/
func TestAssertActorMayRun_ASelfScopedToolNeedsAPersonNotAGrant(t *testing.T) {
	t.Parallel()

	tool := selfScopedTool{&recordingTool{
		name:      "add_home_widget",
		resource:  permission.ResourceHomeLayoutPreset,
		operation: permission.OpUpdate,
	}}
	perms := &fakePermissions{allowed: false}
	executor := newExecutor(tool, &fakeProposalRepo{}, perms)

	person := &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   pulid.MustNew("usr_"),
		UserID:        pulid.MustNew("usr_"),
	}
	require.NoError(t, executor.assertActorMayRun(t.Context(), tool, person))
	assert.Nil(t, perms.lastReq, "no grant is asked for")

	agentActor := &services.RequestActor{
		PrincipalType: services.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("agdef_"),
	}
	require.Error(t, executor.assertActorMayRun(t.Context(), tool, agentActor))
}

// ownedSchemaTool is a self-scoped tool whose schema, like every real one,
// admits nothing it does not declare; the owner the runtime records is not
// among what it declares.
type ownedSchemaTool struct {
	*schemaTool
	validated map[string]any
}

func (t *ownedSchemaTool) Policy() services.ToolPolicy {
	return selfScopedPolicy(t.schemaTool.Policy())
}

func (t *ownedSchemaTool) Validate(_ context.Context, params services.ToolExecuteParams) error {
	t.validated = params.Params

	return nil
}

func ownedProposal(t *testing.T) (*ownedSchemaTool, *agent.AgentProposal, *services.RequestActor) {
	t.Helper()

	tool := &ownedSchemaTool{schemaTool: &schemaTool{}}
	proposal := testProposal(tool.Name(), map[string]any{
		"workerId": "wrk_1",
		"message":  "Call in",
	})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)
	proposal.ToolParams[services.SelfScopeOwnerParam] = actor.UserID.String()

	return tool, proposal, actor
}

func ownerFieldRefused(t *testing.T, err error) {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, services.SelfScopeOwnerParam, multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrForbidden, multiErr.Errors[0].Code)
}

// The owner the runtime stored is not one of the tool's parameters, so the
// schema does not see it; the tool does, exactly as it was stored.
func TestExecute_ApprovesWithChangesAProposalForThePersonsOwnRecords(t *testing.T) {
	t.Parallel()

	tool, proposal, actor := ownedProposal(t)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: false})

	require.NoError(t, executor.Execute(
		t.Context(),
		proposal,
		map[string]any{"priority": "high"},
		actor,
	))

	assert.Equal(t, map[string]any{
		"workerId":                   "wrk_1",
		"message":                    "Call in",
		"priority":                   "high",
		services.SelfScopeOwnerParam: actor.UserID.String(),
	}, tool.ran)
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status)
}

func TestCheckModifications_ChecksAProposalForThePersonsOwnRecords(t *testing.T) {
	t.Parallel()

	tool, proposal, actor := ownedProposal(t)
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: false})

	params, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"priority": "high"},
		actor,
	)

	require.NoError(t, err)
	assert.Equal(t, actor.UserID.String(), params[services.SelfScopeOwnerParam])
	assert.Equal(
		t,
		actor.UserID.String(),
		tool.validated[services.SelfScopeOwnerParam],
		"the tool checks the owner it will run for",
	)
}

// An approver cannot point a change at someone else's records: naming an
// owner is refused on the field, and the tool never sees the name.
func TestExecute_RefusesAChangeToWhoseRecordsItIsFor(t *testing.T) {
	t.Parallel()

	tool, proposal, actor := ownedProposal(t)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: false})
	someoneElse := pulid.MustNew("usr_").String()

	err := executor.Execute(t.Context(), proposal, map[string]any{
		"priority":                   "high",
		services.SelfScopeOwnerParam: someoneElse,
	}, actor)

	ownerFieldRefused(t, err)
	assert.Nil(t, tool.ran, "the tool did not run")
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
	assert.Equal(t, actor.UserID.String(), proposal.ToolParams[services.SelfScopeOwnerParam])
}

func TestCheckModifications_RefusesAChangeToWhoseRecordsItIsFor(t *testing.T) {
	t.Parallel()

	tool, proposal, actor := ownedProposal(t)
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: false})

	_, err := executor.CheckModifications(t.Context(), proposal, map[string]any{
		services.SelfScopeOwnerParam: pulid.MustNew("usr_").String(),
	}, actor)

	ownerFieldRefused(t, err)
	assert.Nil(t, tool.validated, "the tool was never asked about the new owner")
}

func TestExecute_ApprovesAProposalForThePersonsOwnRecordsAsProposed(t *testing.T) {
	t.Parallel()

	tool, proposal, actor := ownedProposal(t)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: false})

	require.NoError(t, executor.Execute(t.Context(), proposal, nil, actor))

	assert.Equal(t, proposal.ToolParams, tool.ran)
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status)
}

// Whatever reaches the merge, the owner that runs is the one stored.
func TestMergeParams_KeepsTheStoredOwner(t *testing.T) {
	t.Parallel()

	stored := pulid.MustNew("usr_").String()
	merged := MergeParams(
		map[string]any{"message": "Call in", services.SelfScopeOwnerParam: stored},
		map[string]any{services.SelfScopeOwnerParam: pulid.MustNew("usr_").String()},
	)
	assert.Equal(t, stored, merged[services.SelfScopeOwnerParam])

	unowned := MergeParams(
		map[string]any{"message": "Call in"},
		map[string]any{services.SelfScopeOwnerParam: stored},
	)
	assert.NotContains(t, unowned, services.SelfScopeOwnerParam)
}
