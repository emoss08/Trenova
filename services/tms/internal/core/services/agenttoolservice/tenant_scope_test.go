package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type approveOnlyTool struct{}

func (approveOnlyTool) Name() string                { return "approve_only" }
func (approveOnlyTool) Description() string         { return "" }
func (approveOnlyTool) ParamSchema() map[string]any { return map[string]any{} }
func (approveOnlyTool) Reversible() bool            { return false }
func (approveOnlyTool) PermissionResource() permission.Resource {
	return permission.ResourceBillingQueue
}
func (approveOnlyTool) PermissionOperation() permission.Operation { return permission.OpApprove }
func (approveOnlyTool) RequiresIdempotencyKey() bool              { return false }
func (approveOnlyTool) DefaultAutonomyTier() agent.AutonomyTier   { return agent.TierPropose }
func (approveOnlyTool) Execute(_ context.Context, _ serviceports.ToolExecuteParams) error {
	return nil
}

func TestGuardExecute_AgentCannotApprove(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	err := guardExecute(approveOnlyTool{}, serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeAgent,
			PrincipalID:    serviceports.AgentPrincipalID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
	})

	require.ErrorIs(t, err, ErrAgentCannotApprove)
}

func TestExecute_TenantMismatch_DoesNotCallPort(t *testing.T) {
	// The mock has no expectations configured; if the tool reaches the port,
	// testify panics, failing the test.
	billing := mocks.NewMockBillingQueueService(t)
	tool := newTransitionToInReviewTool(billing)

	actorOrg := pulid.MustNew("org_")
	actorBu := pulid.MustNew("bu_")

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			OrganizationID: actorOrg,
			BusinessUnitID: actorBu,
		},
		Params: map[string]any{"billingQueueItemId": pulid.MustNew("bqi_").String()},
	})

	require.ErrorIs(t, err, ErrTenantMismatch)
}

type fakeExceptionService struct {
	serviceports.AgentExceptionService

	flagged []*serviceports.FlagAgentExceptionRequest
}

func (f *fakeExceptionService) Flag(
	_ context.Context,
	req *serviceports.FlagAgentExceptionRequest,
	_ *serviceports.RequestActor,
) (*agent.AgentException, error) {
	f.flagged = append(f.flagged, req)
	return &agent.AgentException{ID: pulid.MustNew("aexc_")}, nil
}

func agentActorFor(orgID, buID pulid.ID) *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    serviceports.AgentPrincipalID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

// The move, the driver and the equipment all come from the model; the tenant
// never does. It is taken from the actor so an agent cannot assign a driver
// in another organization by naming that organization's ids.
func TestAssignMove_UsesTheActorTenantAndPassesEveryId(t *testing.T) {
	assignments := mocks.NewMockAssignmentService(t)
	tool := newAssignMoveTool(assignments)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	moveID := pulid.MustNew("smv_")
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trc_")
	trailerID := pulid.MustNew("trl_")

	assignments.EXPECT().
		AssignToMove(mock.Anything, mock.MatchedBy(func(req *repositories.AssignShipmentMoveRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.ShipmentMoveID == moveID &&
				req.PrimaryWorkerID == workerID &&
				req.TractorID == tractorID &&
				req.TrailerID != nil && *req.TrailerID == trailerID &&
				req.SecondaryWorkerID == nil
		})).
		Return(&shipment.Assignment{}, nil)

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agentActorFor(orgID, buID),
		Params: map[string]any{
			"shipmentMoveId":  moveID.String(),
			"primaryWorkerId": workerID.String(),
			"tractorId":       tractorID.String(),
			"trailerId":       trailerID.String(),
		},
	})
	require.NoError(t, err)

	require.Equal(t, permission.ResourceShipmentMove, tool.PermissionResource())
	require.Equal(t, permission.OpUpdate, tool.PermissionOperation())
	require.True(t, tool.Reversible())
}

func TestAssignMove_TenantMismatch_DoesNotCallPort(t *testing.T) {
	assignments := mocks.NewMockAssignmentService(t)
	tool := newAssignMoveTool(assignments)

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Actor:          agentActorFor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
		Params: map[string]any{
			"shipmentMoveId":  pulid.MustNew("smv_").String(),
			"primaryWorkerId": pulid.MustNew("wrk_").String(),
			"tractorId":       pulid.MustNew("trc_").String(),
		},
	})

	require.ErrorIs(t, err, ErrTenantMismatch)
}

// The run an exception belongs to is the one executing the tool, never one
// the model names: a run id in the parameters is ignored.
func TestRaiseException_TiesTheCaseToTheExecutingRun(t *testing.T) {
	exceptions := &fakeExceptionService{}
	tool := newRaiseExceptionTool(exceptions)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	runID := pulid.MustNew("arun_")
	subjectID := pulid.MustNew("bqi_")

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agentActorFor(orgID, buID),
		RunID:          runID,
		Params: map[string]any{
			"runId":          pulid.MustNew("arun_").String(),
			"subjectType":    "BillingQueueItem",
			"subjectId":      subjectID.String(),
			"category":       "MissingDocumentation",
			"severity":       "High",
			"attemptSummary": "Requested the POD twice; no reply.",
			"blastRadius":    float64(1),
		},
	})
	require.NoError(t, err)

	require.Len(t, exceptions.flagged, 1)
	flagged := exceptions.flagged[0]
	require.Equal(t, runID, flagged.RunID)
	require.Equal(t, orgID, flagged.TenantInfo.OrgID)
	require.Equal(t, agent.SubjectBillingQueueItem, flagged.SubjectType)
	require.Equal(t, subjectID, flagged.SubjectID)
	require.Equal(t, agent.CategoryMissingDocumentation, flagged.Category)
	require.Equal(t, agent.SeverityHigh, flagged.Severity)
	require.Equal(t, 1, flagged.BlastRadius)
	require.Equal(t, agent.TierAutoExecute, tool.DefaultAutonomyTier())
}

func TestRaiseException_RefusesWithoutARun(t *testing.T) {
	exceptions := &fakeExceptionService{}
	tool := newRaiseExceptionTool(exceptions)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agentActorFor(orgID, buID),
		Params: map[string]any{
			"subjectType":    "BillingQueueItem",
			"subjectId":      pulid.MustNew("bqi_").String(),
			"category":       "Other",
			"severity":       "Low",
			"attemptSummary": "x",
		},
	})

	require.ErrorIs(t, err, ErrMissingRun)
	require.Empty(t, exceptions.flagged)
}
