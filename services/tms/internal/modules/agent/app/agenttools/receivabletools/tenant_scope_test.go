package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type approveOnlyTool struct{}

func (approveOnlyTool) Name() string { return "approve_only" }

func (approveOnlyTool) Description() string { return "" }

func (approveOnlyTool) ParamSchema() map[string]any { return map[string]any{} }

func (approveOnlyTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          "approve_only",
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBillingQueue,
		Operation:     permission.OpApprove,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "An approval stub.",
	}
}

func (approveOnlyTool) Execute(_ context.Context, _ agenttoolports.ToolExecuteParams) error {
	return nil
}

func TestGuardExecute_AgentCannotApprove(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	err := toolguard.GuardExecute(approveOnlyTool{}, agenttoolports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    permissionports.AgentPrincipalID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
	})

	require.ErrorIs(t, err, toolguard.ErrAgentCannotApprove)
}

func TestExecute_TenantMismatch_DoesNotCallPort(t *testing.T) {
	// The mock has no expectations configured; if the tool reaches the port,
	// testify panics, failing the test.
	billing := billingqueueportsmock.NewMockBillingQueueService(t)
	tool := billingtools.NewTransitionToInReviewTool(billing)

	actorOrg := pulid.MustNew("org_")
	actorBu := pulid.MustNew("bu_")

	err := tool.Execute(t.Context(), agenttoolports.ToolExecuteParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			OrganizationID: actorOrg,
			BusinessUnitID: actorBu,
		},
		Params: map[string]any{"billingQueueItemId": pulid.MustNew("bqi_").String()},
	})

	require.ErrorIs(t, err, toolguard.ErrTenantMismatch)
}

type fakeExceptionService struct {
	agentports.AgentExceptionService

	flagged []*agentports.FlagAgentExceptionRequest
}

func (f *fakeExceptionService) Flag(
	_ context.Context,
	req *agentports.FlagAgentExceptionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentException, error) {
	f.flagged = append(f.flagged, req)
	return &agent.AgentException{ID: pulid.MustNew("aexc_")}, nil
}

// The move, the driver and the equipment all come from the model; the tenant
// never does. It is taken from the actor so an agent cannot assign a driver
// in another organization by naming that organization's ids.
func TestAssignMove_UsesTheActorTenantAndPassesEveryId(t *testing.T) {
	assignments := assignmentportsmock.NewMockAssignmentService(t)
	tool := masterdatatools.NewAssignMoveTool(assignments, nil)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	moveID := pulid.MustNew("smv_")
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trc_")
	trailerID := pulid.MustNew("trl_")

	assignments.EXPECT().
		AssignToMove(mock.Anything, mock.MatchedBy(func(req *assignmentports.AssignShipmentMoveRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.ShipmentMoveID == moveID &&
				req.PrimaryWorkerID == workerID &&
				req.TractorID == tractorID &&
				req.TrailerID != nil && *req.TrailerID == trailerID &&
				req.SecondaryWorkerID == nil
		})).
		Return(&shipment.Assignment{}, nil)

	err := tool.Execute(t.Context(), agenttoolports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agenttoolstest.AgentActorFor(orgID, buID),
		Params: map[string]any{
			"shipmentMoveId":  moveID.String(),
			"primaryWorkerId": workerID.String(),
			"tractorId":       tractorID.String(),
			"trailerId":       trailerID.String(),
		},
	})
	require.NoError(t, err)

	require.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
	require.Equal(t, permission.OpUpdate, tool.Policy().Operation)
	require.True(t, tool.Policy().Reversible)
}

func TestAssignMove_TenantMismatch_DoesNotCallPort(t *testing.T) {
	assignments := assignmentportsmock.NewMockAssignmentService(t)
	tool := masterdatatools.NewAssignMoveTool(assignments, nil)

	err := tool.Execute(t.Context(), agenttoolports.ToolExecuteParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Actor:          agenttoolstest.AgentActorFor(pulid.MustNew("org_"), pulid.MustNew("bu_")),
		Params: map[string]any{
			"shipmentMoveId":  pulid.MustNew("smv_").String(),
			"primaryWorkerId": pulid.MustNew("wrk_").String(),
			"tractorId":       pulid.MustNew("trc_").String(),
		},
	})

	require.ErrorIs(t, err, toolguard.ErrTenantMismatch)
}

// The run an exception belongs to is the one executing the tool, never one
// the model names: a run id in the parameters is ignored.
func TestRaiseException_TiesTheCaseToTheExecutingRun(t *testing.T) {
	exceptions := &fakeExceptionService{}
	tool := NewRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true})

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	runID := pulid.MustNew("arun_")
	subjectID := pulid.MustNew("bqi_")

	err := tool.Execute(t.Context(), agenttoolports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agenttoolstest.AgentActorFor(orgID, buID),
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
	require.Equal(t, agent.TierAutoExecute, tool.Policy().DefaultTier)
}

func TestRaiseException_RefusesWithoutARun(t *testing.T) {
	exceptions := &fakeExceptionService{}
	tool := NewRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true})
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	err := tool.Execute(t.Context(), agenttoolports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agenttoolstest.AgentActorFor(orgID, buID),
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
