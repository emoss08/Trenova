package dispatchautoassignservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePlanner struct {
	plan     *dispatchplanports.Plan
	err      error
	requests []*dispatchplanports.Request
}

func (p *fakePlanner) Plan(
	_ context.Context,
	req *dispatchplanports.Request,
) (*dispatchplanports.Plan, error) {
	p.requests = append(p.requests, req)

	return p.plan, p.err
}

type organizationShadow struct {
	tenantports.AgentControlRepository

	shadow bool
}

func (r organizationShadow) GetOrCreate(
	_ context.Context,
	_ pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	return &agentcontrol.AgentControl{ShadowMode: r.shadow}, nil
}

type dispatchDefinition struct {
	agentdefinitionports.AgentDefinitionRepository

	definition *agentdefinition.Definition
}

func (r dispatchDefinition) GetBySystemKey(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	if r.definition == nil || req.SystemKey != systemKeyDispatchAssignment {
		return nil, errortypes.NewNotFoundError("agent definition not found")
	}

	return r.definition, nil
}

func autoExecuteDefinition() *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:              pulid.MustNew("agd_"),
		Enabled:         true,
		AutonomyCeiling: agent.TierAutoExecute,
		ToolTiers: map[string]agent.AutonomyTier{
			assignMoveTool: agent.TierAutoExecute,
		},
	}
}

type planningRig struct {
	svc       *Service
	planner   *fakePlanner
	proposals *createdProposals
	runs      *inlineRuns
	req       *dispatchconsoleports.DispatchPlanRequest
}

type planningRigParams struct {
	Control            *dispatchcontrol.DispatchControl
	Planned            *dispatchplanports.Plan
	PlannerErr         error
	Definition         *agentdefinition.Definition
	OrganizationShadow bool
}

func newPlanningRig(t *testing.T, p planningRigParams) *planningRig {
	t.Helper()

	req := planRequest()
	controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	controls.EXPECT().
		GetOrCreate(mock.Anything, req.TenantInfo.OrgID, req.TenantInfo.BuID).
		Return(p.Control, nil)

	planner := &fakePlanner{plan: p.Planned, err: p.PlannerErr}
	proposals := &createdProposals{}
	runs := &inlineRuns{}

	svc := New(Params{
		Logger:              zap.NewNop(),
		DispatchControlRepo: controls,
		AgentControlRepo:    organizationShadow{shadow: p.OrganizationShadow},
		DefinitionRepo:      dispatchDefinition{definition: p.Definition},
		ProposalRepo:        proposals,
		Planner:             planner,
		AgentRunService:     runs,
		Executor: proposalexecutor.New(proposalexecutor.Params{
			Logger:       zap.NewNop(),
			Tools:        &agenttest.StubActionRegistry{},
			ProposalRepo: proposals,
		}),
		WorkerRepo: &namedWorkers{},
	})

	return &planningRig{
		svc:       svc,
		planner:   planner,
		proposals: proposals,
		runs:      runs,
		req:       req,
	}
}

func enabledControl() *dispatchcontrol.DispatchControl {
	return &dispatchcontrol.DispatchControl{
		EnableAutoAssignment:          true,
		AutoAssignConfidenceThreshold: decimal.RequireFromString("0.85"),
	}
}

func solvedPlan() *dispatchplanports.Plan {
	confident := pairing(candidate())

	doubtful := candidate()
	doubtful.WorkerName = "Lee Park"
	doubtful.HardCents = 50_000
	unsure := pairing(doubtful)
	unsure.ProNumber = "PRO-43"

	return &dispatchplanports.Plan{
		Mode:        dispatchcontrol.PlanningModeImmediate,
		GeneratedAt: 1_800_000_000,
		Pairings:    []*dispatchplanports.Pairing{confident, unsure},
		Uncovered: []*dispatchplanports.Uncovered{{
			MoveID:             pulid.MustNew("smv_"),
			ProNumber:          "PRO-44",
			Reason:             dispatchplanprice.UncoveredNoCandidate,
			CoveragePriceCents: 40_000,
		}},
		Tours: []*dispatchplanports.Tour{{
			TourID:   pulid.MustNew("tour_"),
			WorkerID: confident.Candidate.WorkerID,
			MoveIDs:  []pulid.ID{confident.MoveID},
		}},
		CostCents:      12_345,
		UncoveredCents: 40_000,
	}
}

func TestPlan_ProposesEveryPairingThePlannerChose(t *testing.T) {
	t.Parallel()

	planned := solvedPlan()
	rig := newPlanningRig(t, planningRigParams{Control: enabledControl(), Planned: planned})
	rig.req.WindowStart = 1_800_000_000
	rig.req.WindowEnd = 1_800_086_400
	rig.req.FleetCodeIDs = []pulid.ID{pulid.MustNew("fc_")}
	rig.req.MoveIDs = []pulid.ID{planned.Pairings[0].MoveID}

	plan, err := rig.svc.Plan(t.Context(), rig.req)
	require.NoError(t, err)

	require.Len(t, rig.planner.requests, 1)
	assert.Equal(t, &dispatchplanports.Request{
		TenantInfo:   rig.req.TenantInfo,
		WindowStart:  rig.req.WindowStart,
		WindowEnd:    rig.req.WindowEnd,
		FleetCodeIDs: rig.req.FleetCodeIDs,
		MoveIDs:      rig.req.MoveIDs,
	}, rig.planner.requests[0])

	assert.Equal(t, dispatchcontrol.PlanningModeImmediate, plan.Mode)
	assert.Equal(t, int64(1_800_000_000), plan.GeneratedAt)
	assert.Equal(t, planned.Uncovered, plan.Uncovered)
	assert.Equal(t, planned.Tours, plan.Tours)
	assert.Equal(t, int64(12_345), plan.CostCents)
	assert.Equal(t, int64(40_000), plan.UncoveredCents)
	assert.False(t, plan.ShadowMode)
	assert.Equal(t, agent.TierPropose, plan.AutonomyTier,
		"without a dispatch agent the plan only proposes")
	assert.False(t, plan.RunID.IsNil())

	require.Len(t, plan.Assignments, 2)
	require.Len(t, rig.proposals.created, 2)
	for i, assignment := range plan.Assignments {
		assert.Same(t, planned.Pairings[i], assignment.Pairing)
		assert.False(t, assignment.AutoExecutable)
		assert.Equal(t, rig.proposals.created[i].ID, assignment.ProposalID)
		assert.Equal(t, assignment.MoveID, rig.proposals.created[i].TargetID)
	}
	assert.True(t, plan.Assignments[0].Confidence.Equal(decimal.NewFromInt(1)))
	assert.True(t, plan.Assignments[1].Confidence.Equal(decimal.RequireFromString("0.5")))
	assert.Empty(t, rig.proposals.executions)
}

func TestPlan_AutoAssignmentDisabledRefusesBeforePlanning(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{
		Control: &dispatchcontrol.DispatchControl{EnableAutoAssignment: false},
		Planned: solvedPlan(),
	})

	plan, err := rig.svc.Plan(t.Context(), rig.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Nil(t, plan)
	assert.Empty(t, rig.planner.requests)
	assert.Empty(t, rig.runs.started)
}

func TestPlan_APlannerFailureIsReturnedAndRecordsNothing(t *testing.T) {
	t.Parallel()

	solveErr := errors.New("snapshot read failed")
	rig := newPlanningRig(t, planningRigParams{
		Control:    enabledControl(),
		PlannerErr: solveErr,
	})

	plan, err := rig.svc.Plan(t.Context(), rig.req)

	require.ErrorIs(t, err, solveErr)
	assert.Nil(t, plan)
	assert.Empty(t, rig.runs.started)
	assert.Empty(t, rig.proposals.created)
}

func TestPlan_AnEmptyPlanRecordsNoRun(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{
		Control: enabledControl(),
		Planned: &dispatchplanports.Plan{Mode: dispatchcontrol.PlanningModeImmediate},
	})

	plan, err := rig.svc.Plan(t.Context(), rig.req)
	require.NoError(t, err)

	assert.Empty(t, plan.Assignments)
	assert.True(t, plan.RunID.IsNil())
	assert.Empty(t, rig.runs.started)
}

func TestPlan_ApplyExecutesOnlyAutoExecutableAssignmentsAndKeepsFailuresPending(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{
		Control:    enabledControl(),
		Planned:    solvedPlan(),
		Definition: autoExecuteDefinition(),
	})
	rig.req.Apply = true

	plan, err := rig.svc.Plan(t.Context(), rig.req)
	require.NoError(t, err)

	assert.Equal(t, agent.TierAutoExecute, plan.AutonomyTier)
	require.Len(t, plan.Assignments, 2)
	require.Len(t, rig.proposals.executions, 1,
		"only the confident assignment is executed; the doubtful one waits for a person")
	execution := rig.proposals.executions[0]
	assert.Equal(t, plan.Assignments[0].ProposalID, execution.ID)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, execution.Status)
	assert.Contains(t, execution.ExecutionError, proposalexecutor.ErrToolMissing.Error())
	assert.False(t, plan.Assignments[0].AutoExecutable,
		"an assignment whose execution failed is reported as still pending")
	assert.False(t, plan.Assignments[1].AutoExecutable)
}

func TestPlan_WithoutApplyNothingExecutes(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{
		Control:    enabledControl(),
		Planned:    solvedPlan(),
		Definition: autoExecuteDefinition(),
	})

	plan, err := rig.svc.Plan(t.Context(), rig.req)
	require.NoError(t, err)

	assert.True(t, plan.Assignments[0].AutoExecutable)
	assert.False(t, plan.Assignments[1].AutoExecutable)
	assert.Empty(t, rig.proposals.executions)
}

func TestPlan_OrganizationShadowModeProposesButNeverExecutes(t *testing.T) {
	t.Parallel()

	rig := newPlanningRig(t, planningRigParams{
		Control:            enabledControl(),
		Planned:            solvedPlan(),
		Definition:         autoExecuteDefinition(),
		OrganizationShadow: true,
	})
	rig.req.Apply = true

	plan, err := rig.svc.Plan(t.Context(), rig.req)
	require.NoError(t, err)

	assert.True(t, plan.ShadowMode)
	require.Len(t, plan.Assignments, 2)
	for _, assignment := range plan.Assignments {
		assert.False(t, assignment.AutoExecutable)
	}
	assert.Len(t, rig.proposals.created, 2)
	assert.Empty(t, rig.proposals.executions)
}

func TestApplyAutoExecutable_ShadowModeExecutesNothing(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t)
	req := planRequest()
	plan := onePlannedMove()
	plan.ShadowMode = true
	plan.Assignments[0].AutoExecutable = true

	recorded, err := svc.recordPlan(t.Context(), req, Policy{Tier: agent.TierAutoExecute}, plan)
	require.NoError(t, err)

	svc.applyAutoExecutable(t.Context(), req, plan, recorded)

	assert.Empty(t, proposals.executions)
	assert.True(t, plan.Assignments[0].AutoExecutable)
}

func TestApplyAutoExecutable_SkipsAssignmentsThatMayNotAutoExecute(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t)
	req := planRequest()
	plan := onePlannedMove()

	recorded, err := svc.recordPlan(t.Context(), req, Policy{Tier: agent.TierAutoExecute}, plan)
	require.NoError(t, err)

	svc.applyAutoExecutable(t.Context(), req, plan, recorded)

	assert.Empty(t, proposals.executions)
}

func TestApplyAutoExecutable_AMissingProposalIsNotExecuted(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t)
	plan := onePlannedMove()
	plan.Assignments[0].AutoExecutable = true

	svc.applyAutoExecutable(t.Context(), planRequest(), plan, nil)

	assert.Empty(t, proposals.executions)
	assert.True(t, plan.Assignments[0].AutoExecutable)
}
