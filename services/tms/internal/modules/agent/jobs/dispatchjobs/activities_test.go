package dispatchjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchconsoleportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

type stubProposalRepo struct {
	mock.Mock
}

func (m *stubProposalRepo) ExpirePendingByRun(
	ctx context.Context,
	req agentproposalports.ExpireAgentProposalsByRunRequest,
) (int, error) {
	args := m.Called(ctx, req)
	return args.Int(0), args.Error(1)
}

func (m *stubProposalRepo) ExpirePendingByDefinition(
	context.Context,
	agentproposalports.ExpireAgentProposalsByDefinitionRequest,
) (int, error) {
	return 0, nil
}

func (m *stubProposalRepo) ExpirePending(
	context.Context,
	agentproposalports.ExpireAgentProposalsRequest,
) (int, error) {
	return 0, nil
}

func (m *stubProposalRepo) ListByIDs(
	context.Context,
	agentproposalports.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) ListPendingForReminder(
	context.Context,
	agentproposalports.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) MarkReminded(
	context.Context,
	agentproposalports.MarkProposalsRemindedRequest,
) (int, error) {
	return 0, nil
}

func (m *stubProposalRepo) List(
	context.Context,
	*agentproposalports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return nil, nil
}

func (m *stubProposalRepo) ListConnection(
	context.Context,
	*agentproposalports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return nil, nil
}

func (m *stubProposalRepo) GetByID(
	context.Context,
	agentproposalports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) Create(
	context.Context,
	*agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) RecordExecution(
	context.Context,
	agentproposalports.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) ListByThread(
	context.Context,
	agentproposalports.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) UpdateStatus(
	context.Context,
	agentproposalports.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

type activityMocks struct {
	controls   *dispatchcontrolportsmock.MockDispatchControlRepository
	proposals  *stubProposalRepo
	autoAssign *dispatchconsoleportsmock.MockDispatchAutoAssignService
}

func newActivities(t *testing.T) (*Activities, *activityMocks) {
	t.Helper()

	deps := &activityMocks{
		controls:   dispatchcontrolportsmock.NewMockDispatchControlRepository(t),
		proposals:  new(stubProposalRepo),
		autoAssign: dispatchconsoleportsmock.NewMockDispatchAutoAssignService(t),
	}

	return NewActivities(ActivitiesParams{
		DispatchControlRepo: deps.controls,
		ProposalRepo:        deps.proposals,
		AutoAssign:          deps.autoAssign,
		Logger:              zap.NewNop(),
	}), deps
}

func planWithRun(
	assignments, uncovered int,
	tours ...*dispatchconsoleports.DispatchTour,
) *dispatchconsoleports.DispatchPlan {
	return &dispatchconsoleports.DispatchPlan{
		RunID:       pulid.MustNew("arun_"),
		Assignments: make([]*dispatchconsoleports.DispatchPlannedAssignment, assignments),
		Uncovered:   make([]*dispatchconsoleports.DispatchUncoveredMove, uncovered),
		Tours:       tours,
	}
}

func tourOf(moves int, deadhead float64) *dispatchconsoleports.DispatchTour {
	moveIDs := make([]pulid.ID, 0, moves)
	for range moves {
		moveIDs = append(moveIDs, pulid.MustNew("mov_"))
	}

	return &dispatchconsoleports.DispatchTour{
		TourID:             pulid.MustNew("tour_"),
		WorkerID:           pulid.MustNew("wrk_"),
		MoveIDs:            moveIDs,
		TotalDeadheadMiles: deadhead,
	}
}

func TestHorizonPlanSweepActivity_SummarisesEachTenant(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	first, second := tenant(), tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{first, second}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == first.OrgID
		})).
		Return(&dispatchconsoleports.DispatchPlan{
			RunID:       pulid.MustNew("arun_"),
			Assignments: make([]*dispatchconsoleports.DispatchPlannedAssignment, 5),
			Uncovered:   make([]*dispatchconsoleports.DispatchUncoveredMove, 2),
			Tours:       []*dispatchconsoleports.DispatchTour{tourOf(3, 40), tourOf(2, 25)},
			TotalScore:  400,
			ShadowMode:  true,
		}, nil)

	deps.proposals.On("ExpirePendingByRun", mock.Anything, mock.Anything).Return(5, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == second.OrgID
		})).
		Return(&dispatchconsoleports.DispatchPlan{
			Assignments: make([]*dispatchconsoleports.DispatchPlannedAssignment, 1),
			Uncovered:   make([]*dispatchconsoleports.DispatchUncoveredMove, 0),
			Tours:       []*dispatchconsoleports.DispatchTour{tourOf(1, 10)},
			TotalScore:  90,
		}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 2, result.TenantsScanned)
	assert.Equal(t, 2, result.TenantsPlanned)
	assert.Zero(t, result.TenantsFailed)
	assert.Equal(t, 6, result.MovesPlanned)
	assert.Equal(t, 2, result.MovesUncovered)
	assert.Equal(t, 3, result.ToursBuilt)

	assert.Equal(t, 3, result.ChainedMoves)

	require.Len(t, result.TenantOutcomes, 2)
	assert.InDelta(t, 65.0, result.TenantOutcomes[0].TotalDeadheadMiles, 0.001)
	assert.True(t, result.TenantOutcomes[0].ShadowMode)
	assert.Equal(t, 5, result.TenantOutcomes[0].ProposalsRetired)
}

func TestHorizonPlanSweepActivity_RetiresItsOwnProposals(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	only := tenant()
	plan := planWithRun(4, 0, tourOf(4, 30))

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{only}, nil)

	deps.autoAssign.EXPECT().Plan(mock.Anything, mock.Anything).Return(plan, nil)

	deps.proposals.On(
		"ExpirePendingByRun",
		mock.Anything,
		mock.MatchedBy(func(req agentproposalports.ExpireAgentProposalsByRunRequest) bool {
			return req.RunID == plan.RunID && req.TenantInfo.OrgID == only.OrgID
		}),
	).Return(4, nil).Once()

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	require.Len(t, result.TenantOutcomes, 1)
	assert.Equal(t, 4, result.TenantOutcomes[0].ProposalsRetired)
	assert.Equal(t, plan.RunID.String(), result.TenantOutcomes[0].RunID)
}

func TestHorizonPlanSweepActivity_RetirementFailureDoesNotFailTheTenant(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	only := tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{only}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.Anything).
		Return(planWithRun(3, 1, tourOf(3, 20)), nil)

	deps.proposals.On("ExpirePendingByRun", mock.Anything, mock.Anything).
		Return(0, errors.New("deadlock detected"))

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, result.TenantsPlanned)
	assert.Zero(t, result.TenantsFailed)
	assert.Equal(t, 3, result.MovesPlanned)
	assert.Zero(t, result.TenantOutcomes[0].ProposalsRetired)
}

func TestHorizonPlanSweepActivity_SkipsRetirementWithoutARun(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{tenant()}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.Anything).
		Return(&dispatchconsoleports.DispatchPlan{}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, result.TenantsPlanned)
	deps.proposals.AssertNotCalled(t, "ExpirePendingByRun", mock.Anything, mock.Anything)
}

func TestHorizonPlanSweepActivity_NeverApplies(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	only := tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{only}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.DispatchPlanRequest) bool {
			return !req.Apply
		})).
		Return(&dispatchconsoleports.DispatchPlan{}, nil)

	_, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)
}

func TestHorizonPlanSweepActivity_OneTenantFailingDoesNotStopTheSweep(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	broken, healthy := tenant(), tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{broken, healthy}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == broken.OrgID
		})).
		Return(nil, errors.New("Auto assignment is disabled for this organization"))

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == healthy.OrgID
		})).
		Return(&dispatchconsoleports.DispatchPlan{
			Assignments: make([]*dispatchconsoleports.DispatchPlannedAssignment, 2),
			Tours:       []*dispatchconsoleports.DispatchTour{tourOf(2, 15)},
		}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 2, result.TenantsScanned)
	assert.Equal(t, 1, result.TenantsPlanned)
	assert.Equal(t, 1, result.TenantsFailed)
	assert.Equal(t, 2, result.MovesPlanned)

	require.Len(t, result.TenantOutcomes, 2)
	assert.Contains(t, result.TenantOutcomes[0].Error, "disabled")
	assert.Empty(t, result.TenantOutcomes[1].Error)
}

func TestHorizonPlanSweepActivity_NoTenantsIsNotAnError(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Zero(t, result.TenantsScanned)
	assert.Empty(t, result.TenantOutcomes)
}

func TestHorizonPlanSweepActivity_PropagatesTenantLookupFailure(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return(nil, errors.New("connection refused"))

	result, err := activities.HorizonPlanSweepActivity(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
}

func (s *stubProposalRepo) RecordSimulation(
	context.Context,
	agentproposalports.RecordAgentProposalSimulationRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (s *stubProposalRepo) SetPendingModifications(
	context.Context,
	agentproposalports.SetPendingModificationsRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (s *stubProposalRepo) CountExecutedTool(
	context.Context,
	agentproposalports.CountExecutedToolRequest,
) (int, error) {
	return 0, nil
}

func (s *stubProposalRepo) ListByPlans(
	context.Context,
	agentproposalports.ListAgentProposalsByPlansRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (s *stubProposalRepo) ListByPlan(
	context.Context,
	agentproposalports.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (s *stubProposalRepo) SkipPendingByPlan(
	context.Context,
	agentproposalports.SkipPendingByPlanRequest,
) (int, error) {
	return 0, nil
}

func (s *stubProposalRepo) ListByRun(
	context.Context,
	agentproposalports.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) ExpirePendingByThread(
	context.Context,
	agentproposalports.ExpireAgentProposalsByThreadRequest,
) (int, error) {
	return 0, nil
}

func (s *stubProposalRepo) SumAutomaticMoney(
	context.Context,
	*agentproposalports.SumAutomaticMoneyRequest,
) (agenttooltrust.MoneyUse, error) {
	return agenttooltrust.MoneyUse{}, nil
}
