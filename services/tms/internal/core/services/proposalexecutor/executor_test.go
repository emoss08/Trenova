package proposalexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingTool struct {
	name        string
	resource    permission.Resource
	operation   permission.Operation
	err         error
	calls       int
	lastParams  services.ToolExecuteParams
	requiresKey bool
}

func (t *recordingTool) Name() string                              { return t.name }
func (t *recordingTool) Description() string                       { return "recording tool" }
func (t *recordingTool) ParamSchema() map[string]any               { return map[string]any{} }
func (t *recordingTool) Reversible() bool                          { return true }
func (t *recordingTool) PermissionResource() permission.Resource   { return t.resource }
func (t *recordingTool) PermissionOperation() permission.Operation { return t.operation }
func (t *recordingTool) RequiresIdempotencyKey() bool              { return t.requiresKey }
func (t *recordingTool) DefaultAutonomyTier() agent.AutonomyTier   { return agent.TierPropose }

func (t *recordingTool) Execute(_ context.Context, params services.ToolExecuteParams) error {
	t.calls++
	t.lastParams = params

	return t.err
}

type fakeRegistry struct{ tools []services.AgentTool }

func (r fakeRegistry) Get(name string) (services.AgentTool, bool) {
	for _, tool := range r.tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r fakeRegistry) All() []services.AgentTool                   { return r.tools }
func (r fakeRegistry) Descriptors() []services.AgentToolDescriptor { return nil }

type recordedExecution struct {
	status     agent.ProposalStatus
	executedAt *int64
	errText    string
}

type fakeProposalRepo struct {
	recorded []recordedExecution
}

func (r *fakeProposalRepo) RecordExecution(
	_ context.Context,
	req repositories.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	r.recorded = append(r.recorded, recordedExecution{
		status:     req.Status,
		executedAt: req.ExecutedAt,
		errText:    req.ExecutionError,
	})

	return nil, nil
}

type fakePermissions struct {
	allowed  bool
	err      error
	lastReq  *services.PermissionCheckRequest
	checkErr error
}

func (p *fakePermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	p.lastReq = req
	if p.checkErr != nil {
		return nil, p.checkErr
	}

	return &services.PermissionCheckResult{Allowed: p.allowed}, p.err
}

type noopAudit struct{}

func (noopAudit) LogAction(_ *services.LogActionParams, _ ...services.LogOption) error {
	return nil
}

func newExecutor(
	tool services.AgentTool,
	repo *fakeProposalRepo,
	perms *fakePermissions,
) *Service {
	tools := []services.AgentTool{}
	if tool != nil {
		tools = append(tools, tool)
	}

	return &Service{
		l:            zap.NewNop(),
		tools:        fakeRegistry{tools: tools},
		proposalRepo: repo,
		permissions:  perms,
		audit:        noopAudit{},
	}
}

func testProposal(toolName string, params map[string]any) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ToolName:       toolName,
		ToolParams:     params,
		Status:         agent.ProposalStatusAccepted,
	}
}

func testActor(orgID, buID pulid.ID) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func TestExecute_RunsTheToolAndRecordsSuccess(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	proposal := testProposal("reassign_move", map[string]any{"moveId": "mv_1"})

	err := newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, tool.calls)
	assert.Equal(t, "mv_1", tool.lastParams.Params["moveId"])

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecuted, repo.recorded[0].status)
	assert.NotNil(t, repo.recorded[0].executedAt)
	assert.Empty(t, repo.recorded[0].errText)
}

// The approver is the actor, so the write is attributed to the person who
// authorized it rather than to the agent.
func TestExecute_AttributesTheWriteToTheApprover(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	proposal := testProposal("reassign_move", map[string]any{})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)

	require.NoError(t, newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true}).
		Execute(t.Context(), proposal, nil, actor))

	assert.Equal(t, actor.UserID, tool.lastParams.Actor.UserID)
	assert.Equal(t, proposal.OrganizationID, tool.lastParams.OrganizationID)
	assert.Equal(t, proposal.BusinessUnitID, tool.lastParams.BusinessUnitID)
}

// The proposal id is stable across retries of the same approval and distinct
// between proposals, which is what a tool guarding double execution needs.
func TestExecute_UsesTheProposalIdAsIdempotencyKey(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:        "reassign_move",
		resource:    permission.ResourceShipmentMove,
		operation:   permission.OpUpdate,
		requiresKey: true,
	}
	proposal := testProposal("reassign_move", map[string]any{})

	require.NoError(t, newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true}).
		Execute(t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID)))

	assert.Equal(t, proposal.ID.String(), tool.lastParams.IdempotencyKey)
}

// Approving with changes must run what the approver agreed to, not what the
// model originally asked for.
func TestExecute_AppliesApproverModifications(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	proposal := testProposal("reassign_move", map[string]any{
		"moveId":   "mv_1",
		"workerId": "wrk_original",
	})

	require.NoError(t, newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true}).
		Execute(
			t.Context(),
			proposal,
			map[string]any{"workerId": "wrk_replacement"},
			testActor(proposal.OrganizationID, proposal.BusinessUnitID),
		))

	assert.Equal(t, "wrk_replacement", tool.lastParams.Params["workerId"])
	assert.Equal(t, "mv_1", tool.lastParams.Params["moveId"], "unmodified params must survive")
}

// The permission checked is the tool's own. Checking a fixed resource was right
// when the billing agent was the only agent and wrong as soon as a second
// existed.
func TestExecute_ChecksTheToolsOwnPermission(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "update_customer",
		resource:  permission.ResourceCustomer,
		operation: permission.OpUpdate,
	}
	perms := &fakePermissions{allowed: true}
	proposal := testProposal("update_customer", map[string]any{})

	require.NoError(t, newExecutor(tool, &fakeProposalRepo{}, perms).
		Execute(t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID)))

	require.NotNil(t, perms.lastReq)
	assert.Equal(t, permission.ResourceCustomer.String(), perms.lastReq.Resource)
	assert.Equal(t, permission.OpUpdate, perms.lastReq.Operation)
}

func TestExecute_RefusesWhenTheApproverLacksPermission(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	proposal := testProposal("reassign_move", map[string]any{})

	err := newExecutor(tool, repo, &fakePermissions{allowed: false}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.Error(t, err)

	assert.Zero(t, tool.calls, "the tool must not run without permission")
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}

func TestExecute_RecordsFailureWhenTheToolErrors(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "reassign_move",
		resource:  permission.ResourceShipmentMove,
		operation: permission.OpUpdate,
		err:       errors.New("move already delivered"),
	}
	repo := &fakeProposalRepo{}
	proposal := testProposal("reassign_move", map[string]any{})

	err := newExecutor(tool, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.Error(t, err)

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
	// The reason is stored so an approver can see what went wrong without
	// reading logs.
	assert.Contains(t, repo.recorded[0].errText, "move already delivered")
	assert.Nil(t, repo.recorded[0].executedAt)
}

// A proposal can outlive the tool it names — a deployment removes one, and an
// old pending proposal still references it.
func TestExecute_RecordsFailureWhenTheToolNoLongerExists(t *testing.T) {
	t.Parallel()

	repo := &fakeProposalRepo{}
	proposal := testProposal("tool_that_was_removed", map[string]any{})

	err := newExecutor(nil, repo, &fakePermissions{allowed: true}).Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)
	require.ErrorIs(t, err, ErrToolMissing)

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}
