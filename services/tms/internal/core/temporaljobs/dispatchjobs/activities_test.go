package dispatchjobs

import (
	"errors"
	"testing"

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

func newActivities(
	t *testing.T,
) (*Activities, *mocks.MockDispatchControlRepository, *mocks.MockDispatchAutoAssignService) {
	t.Helper()

	controls := mocks.NewMockDispatchControlRepository(t)
	autoAssign := mocks.NewMockDispatchAutoAssignService(t)

	return NewActivities(ActivitiesParams{
		DispatchControlRepo: controls,
		AutoAssign:          autoAssign,
		Logger:              zap.NewNop(),
	}), controls, autoAssign
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

	activities, controls, autoAssign := newActivities(t)
	first, second := tenant(), tenant()

	controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{first, second}, nil)

	autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == first.OrgID
		})).
		Return(&portservices.DispatchPlan{
			Assignments: make([]*portservices.DispatchPlannedAssignment, 5),
			Uncovered:   make([]*portservices.DispatchUncoveredMove, 2),
			Tours:       []*portservices.DispatchTour{tourOf(3, 40), tourOf(2, 25)},
			TotalScore:  400,
			ShadowMode:  true,
		}, nil)

	autoAssign.EXPECT().
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
}

// A scheduled pass exists to gather evidence. If it ever applied, an organization
// still evaluating the planner would find loads assigned out from under it.
func TestHorizonPlanSweepActivity_NeverApplies(t *testing.T) {
	t.Parallel()

	activities, controls, autoAssign := newActivities(t)
	only := tenant()

	controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{only}, nil)

	autoAssign.EXPECT().
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

	activities, controls, autoAssign := newActivities(t)
	broken, healthy := tenant(), tenant()

	controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{broken, healthy}, nil)

	autoAssign.EXPECT().
		Plan(mock.Anything, mock.MatchedBy(func(req *portservices.DispatchPlanRequest) bool {
			return req.TenantInfo.OrgID == broken.OrgID
		})).
		Return(nil, errors.New("Auto assignment is disabled for this organization"))

	autoAssign.EXPECT().
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

	activities, controls, _ := newActivities(t)

	controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return([]pagination.TenantInfo{}, nil)

	result, err := activities.HorizonPlanSweepActivity(t.Context())
	require.NoError(t, err)

	assert.Zero(t, result.TenantsScanned)
	assert.Empty(t, result.TenantOutcomes)
}

func TestHorizonPlanSweepActivity_PropagatesTenantLookupFailure(t *testing.T) {
	t.Parallel()

	activities, controls, _ := newActivities(t)

	controls.EXPECT().
		ListHorizonPlanningTenants(mock.Anything).
		Return(nil, errors.New("connection refused"))

	result, err := activities.HorizonPlanSweepActivity(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
}
