package dispatchjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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

// stubProposalRepo implements the proposal repository for the sweep's sake only. The
// sweep touches one method; the rest exist to satisfy the interface and return zero
// values so an unexpected call shows up as an empty result rather than a panic.
type stubProposalRepo struct {
	mock.Mock
}

func (m *stubProposalRepo) ExpirePendingByRun(
	ctx context.Context,
	req repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	args := m.Called(ctx, req)
	return args.Int(0), args.Error(1)
}

func (m *stubProposalRepo) List(
	context.Context,
	*repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return nil, nil
}

func (m *stubProposalRepo) ListConnection(
	context.Context,
	*repositories.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return nil, nil
}

func (m *stubProposalRepo) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) Create(
	context.Context,
	*agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (m *stubProposalRepo) UpdateStatus(
	context.Context,
	repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

type activityMocks struct {
	controls   *mocks.MockDispatchControlRepository
	proposals  *stubProposalRepo
	autoAssign *mocks.MockDispatchAutoAssignService
}

func newActivities(t *testing.T) (*Activities, *activityMocks) {
	t.Helper()

	deps := &activityMocks{
		controls:   mocks.NewMockDispatchControlRepository(t),
		proposals:  new(stubProposalRepo),
		autoAssign: mocks.NewMockDispatchAutoAssignService(t),
	}

	return NewActivities(ActivitiesParams{
		DispatchControlRepo: deps.controls,
		ProposalRepo:        deps.proposals,
		AutoAssign:          deps.autoAssign,
		Logger:              zap.NewNop(),
	}), deps
}

func planWithRun(assignments, uncovered int, tours ...*portservices.DispatchTour) *portservices.DispatchPlan {
	return &portservices.DispatchPlan{
		RunID:       pulid.MustNew("arun_"),
		Assignments: make([]*portservices.DispatchPlannedAssignment, assignments),
		Uncovered:   make([]*portservices.DispatchUncoveredMove, uncovered),
		Tours:       tours,
	}
}

func tourOf(moves int, deadhead float64) *portservices.DispatchTour {
	moveIDs := make([]pulid.ID, 0, moves)
	for range moves {
		moveIDs = append(moveIDs, pulid.MustNew("mov_"))
	}

	return &portservices.DispatchTour{
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
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == first.OrgID
		})).
		Return(&portservices.DispatchPlan{
			RunID:       pulid.MustNew("arun_"),
			Assignments: make([]*portservices.DispatchPlannedAssignment, 5),
			Uncovered:   make([]*portservices.DispatchUncoveredMove, 2),
			Tours:       []*portservices.DispatchTour{tourOf(3, 40), tourOf(2, 25)},
			TotalScore:  400,
			ShadowMode:  true,
		}, nil)

	deps.proposals.On("ExpirePendingByRun", mock.Anything, mock.Anything).Return(5, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == second.OrgID
		})).
		Return(&portservices.DispatchPlan{
			Assignments: make([]*portservices.DispatchPlannedAssignment, 1),
			Uncovered:   make([]*portservices.DispatchUncoveredMove, 0),
			Tours:       []*portservices.DispatchTour{tourOf(1, 10)},
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

	// Only moves past the first in a tour were genuinely chained: 2 from the
	// three-move tour, 1 from the two-move tour, 0 from the single-move tour.
	assert.Equal(t, 3, result.ChainedMoves)

	require.Len(t, result.TenantOutcomes, 2)
	assert.InDelta(t, 65.0, result.TenantOutcomes[0].TotalDeadheadMiles, 0.001)
	assert.True(t, result.TenantOutcomes[0].ShadowMode)
	assert.Equal(t, 5, result.TenantOutcomes[0].ProposalsRetired)
}

// Planning writes a pending proposal per assignment. Left alone, a half-hourly sweep
// would republish the whole board into the dispatcher's review queue every pass, so
// the sweep retires what it just wrote.
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
		mock.MatchedBy(func(req repositories.ExpireAgentProposalsByRunRequest) bool {
			return req.RunID == plan.RunID && req.TenantInfo.OrgID == only.OrgID
		}),
	).Return(4, nil).Once()

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	require.Len(t, result.TenantOutcomes, 1)
	assert.Equal(t, 4, result.TenantOutcomes[0].ProposalsRetired)
	assert.Equal(t, plan.RunID.String(), result.TenantOutcomes[0].RunID)
}

// The plan is already recorded by the time proposals are retired, so losing the
// retirement should not throw away the tenant's outcome.
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

// A plan with nothing in it never opened a run, so there is nothing to retire.
func TestHorizonPlanSweepActivity_SkipsRetirementWithoutARun(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{tenant()}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.Anything).
		Return(&portservices.DispatchPlan{}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, result.TenantsPlanned)
	deps.proposals.AssertNotCalled(t, "ExpirePendingByRun", mock.Anything, mock.Anything)
}

// A scheduled pass exists to gather evidence. If it ever applied, an organization
// still evaluating the planner would find loads assigned out from under it.
func TestHorizonPlanSweepActivity_NeverApplies(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	only := tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{only}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return !req.Apply
		})).
		Return(&portservices.DispatchPlan{}, nil)

	_, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)
}

// One organization's bad configuration must not stop the rest of the fleet being
// planned.
func TestHorizonPlanSweepActivity_OneTenantFailingDoesNotStopTheSweep(t *testing.T) {
	t.Parallel()

	activities, deps := newActivities(t)
	broken, healthy := tenant(), tenant()

	deps.controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{broken, healthy}, nil)

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == broken.OrgID
		})).
		Return(nil, errors.New("Auto assignment is disabled for this organization"))

	deps.autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == healthy.OrgID
		})).
		Return(&portservices.DispatchPlan{
			Assignments: make([]*portservices.DispatchPlannedAssignment, 2),
			Tours:       []*portservices.DispatchTour{tourOf(2, 15)},
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
