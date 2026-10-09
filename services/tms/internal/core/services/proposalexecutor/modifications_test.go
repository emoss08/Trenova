package proposalexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaTool declares a bounded parameter and, when asked, checks its own
// arguments the way a tool with a validation hook does.
type schemaTool struct {
	validateErr error
	ran         map[string]any
}

func (t *schemaTool) Name() string        { return "notify_driver" }
func (t *schemaTool) Description() string { return "Message a driver." }
func (t *schemaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{"type": "string"},
			"message":  map[string]any{"type": "string", "maxLength": 20},
			"priority": map[string]any{"type": "string", "enum": []string{"low", "high"}},
			"repeat":   map[string]any{"type": "integer"},
			"sendAt":   map[string]any{"type": "string", "format": "local-date-time"},
		},
		"required":             []string{"workerId", "message"},
		"additionalProperties": false,
	}
}
func (t *schemaTool) Policy() services.ToolPolicy {
	return services.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverMessage,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "A schema stub.",
	}
}
func (t *schemaTool) Execute(_ context.Context, params services.ToolExecuteParams) error {
	t.ran = params.Params

	return nil
}
func (t *schemaTool) Validate(context.Context, services.ToolExecuteParams) error {
	return t.validateErr
}

func TestCheckModifications_ReturnsTheParametersAsTheyWouldRun(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	params, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"priority": "high"},
		actor,
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		map[string]any{"workerId": "wrk_1", "message": "Call in", "priority": "high"},
		params,
	)
}

// A change that does not fit the tool is refused as a field error, before
// any decision is recorded, so the form can say which value is wrong.
func TestCheckModifications_RefusesAChangeTheSchemaRejects(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	_, err := executor.CheckModifications(t.Context(), proposal, map[string]any{
		"message":  "this message is far longer than the tool allows",
		"priority": "urgent",
	}, actor)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		fields = append(fields, entry.Field)
	}
	assert.ElementsMatch(t, []string{"message", "priority"}, fields)
}

func TestCheckModifications_LetsTheToolCheckItsOwnArguments(t *testing.T) {
	t.Parallel()

	refusal := errors.New("that driver has no portal access")
	tool := &schemaTool{validateErr: refusal}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	_, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"priority": "high"},
		actor,
	)

	require.ErrorIs(t, err, refusal)
}

func TestCheckModifications_RefusesAnApproverFromAnotherOrganization(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)
	actor.OrganizationID = pulid.MustNew("org_")

	_, err := executor.CheckModifications(
		t.Context(),
		proposal,
		map[string]any{"priority": "high"},
		actor,
	)

	require.ErrorIs(t, err, ErrTenantMismatch)
}

// The check at decision time is repeated where the tool runs, so a change
// recorded against one version of a tool cannot run against another that
// no longer takes it.
func TestExecute_RefusesModificationsTheSchemaRejects(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	err := executor.Execute(t.Context(), proposal, map[string]any{"priority": "urgent"}, actor)

	require.Error(t, err)
	assert.Nil(t, tool.ran, "the tool did not run")
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}

// The approval editor sends what a person typed. What the runtime reads for
// a model it reads for the approver too, so "3" for a number, a local time
// written with a space and an enum in another case are what is checked and
// what runs, not a refusal.
func TestCheckModifications_ReadsWhatTheApproverTypedAsTheToolDeclares(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{"workerId": "wrk_1", "message": "Call in"})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	params, err := executor.CheckModifications(t.Context(), proposal, map[string]any{
		"repeat":   "3",
		"sendAt":   "2026-10-01 08:00",
		"priority": "HIGH",
	}, actor)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"workerId": "wrk_1",
		"message":  "Call in",
		"repeat":   float64(3),
		"sendAt":   "2026-10-01T08:00",
		"priority": "high",
	}, params)
}

// What runs is what was read, whether the value came from the approver or
// from a proposal filed before its parameter's shape was asserted.
func TestExecute_RunsTheParametersAsTheToolDeclaresThem(t *testing.T) {
	t.Parallel()

	tool := &schemaTool{}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	proposal := testProposal(tool.Name(), map[string]any{
		"workerId": "wrk_1",
		"message":  "Call in",
		"sendAt":   "2026-10-01 08:00:30",
	})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	err := executor.Execute(t.Context(), proposal, map[string]any{"repeat": "2"}, actor)

	require.NoError(t, err)
	require.NotNil(t, tool.ran)
	assert.InDelta(t, 2, tool.ran["repeat"], 0)
	assert.Equal(t, "2026-10-01T08:00", tool.ran["sendAt"])
}

// A self-scoped call's owner is never read or renamed, and comes back as
// it was stored.
func TestCoerceParams_LeavesTheOwnerAsStored(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_").String()
	params := CoerceParams(&schemaTool{}, map[string]any{
		"workerId":                   "wrk_1",
		"repeat":                     "4",
		services.SelfScopeOwnerParam: owner,
	})

	assert.Equal(t, owner, params[services.SelfScopeOwnerParam])
	assert.InDelta(t, 4, params["repeat"], 0)
}
