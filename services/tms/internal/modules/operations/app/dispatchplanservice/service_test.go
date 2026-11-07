package dispatchplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchboardportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/integrationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tractorportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const pickupLat, pickupLon = 41.8781, -87.6298

type unroutedLegs struct{}

func (unroutedLegs) EstimateLegs(
	_ context.Context,
	_ pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	return make([]routeestimateports.LegEstimate, len(legs)), nil
}

type planRig struct {
	svc          *Service
	console      *dispatchboardportsmock.MockDispatchConsoleRepository
	controls     *dispatchcontrolportsmock.MockDispatchControlRepository
	integrations *integrationportsmock.MockIntegrationRepository
	tractors     *tractorportsmock.MockTractorRepository
	prices       *dispatchplanportsmock.MockPriceRepository
	tenant       pagination.TenantInfo
	now          int64
}

func newPlanRig(t *testing.T) *planRig {
	t.Helper()

	rig := &planRig{
		console:      dispatchboardportsmock.NewMockDispatchConsoleRepository(t),
		controls:     dispatchcontrolportsmock.NewMockDispatchControlRepository(t),
		integrations: integrationportsmock.NewMockIntegrationRepository(t),
		tractors:     tractorportsmock.NewMockTractorRepository(t),
		prices:       dispatchplanportsmock.NewMockPriceRepository(t),
		tenant: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		now: timeutils.NowUnix(),
	}
	rig.svc = &Service{
		l:           zap.NewNop(),
		consoleRepo: rig.console,
		controls:    rig.controls,
		candidates: dispatchcandidateservice.New(dispatchcandidateservice.Params{
			ConsoleRepo:     rig.console,
			IntegrationRepo: rig.integrations,
			TractorRepo:     rig.tractors,
			RouteEstimator:  unroutedLegs{},
			Logger:          zap.NewNop(),
		}),
		prices: rig.prices,
		clock:  func() int64 { return rig.now },
	}
	rig.controls.EXPECT().
		GetOrCreate(mock.Anything, rig.tenant.OrgID, rig.tenant.BuID).
		Return(dispatchcontrol.NewDefaultDispatchControl(rig.tenant.OrgID, rig.tenant.BuID), nil)

	return rig
}

func (r *planRig) expectMoves(moves []*dispatchboardports.BoardMove, err error) {
	r.console.EXPECT().
		ListBoardMoves(mock.Anything, mock.MatchedBy(func(f *dispatchboardports.DispatchBoardFilter) bool {
			return f.TenantInfo == r.tenant && f.WindowStart == r.now &&
				f.WindowEnd > r.now && f.Limit == maxPlanMoves && f.DriverLimit == maxPlanDrivers
		})).
		Return(moves, err)
}

type rigDriver struct {
	driver  *dispatchboardports.BoardDriver
	worker  *worker.Worker
	tractor *tractor.Tractor
}

func newRigDriver(name string, lat, lon float64) rigDriver {
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trac_")

	return rigDriver{
		driver: &dispatchboardports.BoardDriver{
			WorkerID:  workerID,
			FirstName: name,
			LastName:  "Driver",
			TractorID: tractorID,
		},
		worker: &worker.Worker{
			ID:                   workerID,
			Status:               domaintypes.StatusActive,
			CanBeAssigned:        true,
			AvailableForDispatch: true,
		},
		tractor: &tractor.Tractor{
			ID:                         tractorID,
			Code:                       name,
			Status:                     domaintypes.EquipmentStatusAvailable,
			LastKnownLocationLatitude:  new(lat),
			LastKnownLocationLongitude: new(lon),
		},
	}
}

func (r *planRig) expectStanding(prices ...*dispatchplanprice.DispatchPlanPrice) {
	r.prices.EXPECT().
		ListByMoves(mock.Anything, mock.Anything).
		Return(prices, nil).
		Once()
}

func (r *planRig) expectFleet(drivers ...rigDriver) {
	board := make([]*dispatchboardports.BoardDriver, 0, len(drivers))
	workers := make([]*worker.Worker, 0, len(drivers))
	tractors := make([]*tractor.Tractor, 0, len(drivers))
	for _, d := range drivers {
		board = append(board, d.driver)
		workers = append(workers, d.worker)
		tractors = append(tractors, d.tractor)
	}

	r.console.EXPECT().
		ListBoardDrivers(mock.Anything, mock.Anything).
		Return(&dispatchboardports.BoardDriverPage{Drivers: board}, nil)
	r.integrations.EXPECT().ListByTenant(mock.Anything, r.tenant).Return(nil, nil)
	r.console.EXPECT().ListWorkerCommitments(mock.Anything, mock.Anything).Return(nil, nil)
	r.console.EXPECT().ListWorkerTimeOff(mock.Anything, mock.Anything).Return(nil, nil)
	r.console.EXPECT().ListWorkerWorkload(mock.Anything, mock.Anything).Return(nil, nil)
	r.console.EXPECT().
		ListWorkersByIDs(mock.Anything, r.tenant, mock.Anything).
		Return(workers, nil)
	r.tractors.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(tractors, nil)
	r.console.EXPECT().ListEquipmentByIDs(mock.Anything, mock.Anything).Return(nil, nil, nil)
	r.console.EXPECT().ListWorkerOnTimeStats(mock.Anything, mock.Anything).Return(nil, nil)
	r.console.EXPECT().ListWorkerAcceptanceStats(mock.Anything, mock.Anything).Return(nil, nil)
}

func (r *planRig) move(proNumber string) *dispatchboardports.BoardMove {
	return &dispatchboardports.BoardMove{
		MoveID:            pulid.MustNew("smv_"),
		ProNumber:         proNumber,
		Distance:          new(200.0),
		OriginLatitude:    new(pickupLat),
		OriginLongitude:   new(pickupLon),
		OriginWindowStart: r.now + 3*3600,
	}
}

func TestPlan_ImmediateModeRecordsPricesForEveryMoveItConsidered(t *testing.T) {
	t.Parallel()

	rig := newPlanRig(t)
	near := newRigDriver("Near", pickupLat+0.1, pickupLon)
	far := newRigDriver("Far", pickupLat+1, pickupLon)
	covered := rig.move("PRO-COVERED")
	unplaced := rig.move("PRO-UNPLACED")
	unplaced.OriginLatitude = nil
	unplaced.OriginLongitude = nil

	rig.expectMoves([]*dispatchboardports.BoardMove{covered, unplaced}, nil)
	rig.expectFleet(near, far)
	rig.expectStanding()

	var written *dispatchplanports.ReplacePricesRequest
	rig.prices.EXPECT().
		Replace(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *dispatchplanports.ReplacePricesRequest) error {
			written = req
			return nil
		}).
		Once()

	plan, err := rig.svc.Plan(t.Context(), &dispatchplanports.Request{TenantInfo: rig.tenant})

	require.NoError(t, err)
	assert.Equal(t, dispatchcontrol.PlanningModeImmediate, plan.Mode)
	assert.Equal(t, rig.now, plan.GeneratedAt)
	require.Len(t, plan.Pairings, 1)
	assert.Equal(t, covered.MoveID, plan.Pairings[0].MoveID)
	assert.Equal(t, near.driver.WorkerID, plan.Pairings[0].Candidate.WorkerID)
	require.Len(t, plan.Uncovered, 1)
	assert.Equal(t, unplaced.MoveID, plan.Uncovered[0].MoveID)
	assert.Equal(t, dispatchplanprice.UncoveredLocationUnknown, plan.Uncovered[0].Reason)

	require.NotNil(t, written)
	assert.Equal(t, rig.tenant, written.TenantInfo)
	assert.Equal(t, rig.now, written.GeneratedAt, "prices are stamped with the plan's time")
	assert.Equal(t, []pulid.ID{covered.MoveID, unplaced.MoveID}, written.MoveIDs,
		"every move the plan read is replaced, covered or not")
	require.Len(t, written.Prices, 2)
	for _, price := range written.Prices {
		assert.Equal(t, rig.tenant.OrgID, price.OrganizationID)
		assert.Equal(t, rig.tenant.BuID, price.BusinessUnitID)
		assert.Equal(t, rig.now, price.GeneratedAt)
	}
	assert.Equal(t, covered.MoveID, written.Prices[0].MoveID)
	assert.Equal(t, near.driver.WorkerID, written.Prices[0].WorkerID)
	assert.Equal(t, plan.Pairings[0].Alternatives, written.Prices[0].Alternatives)
	assert.Equal(t, unplaced.MoveID, written.Prices[1].MoveID)
	assert.True(t, written.Prices[1].WorkerID.IsNil())
	assert.Equal(t, dispatchplanprice.UncoveredLocationUnknown, written.Prices[1].UncoveredReason)
}

func TestPlan_EmptyBoardReadsNoFleetAndWritesNoPrices(t *testing.T) {
	t.Parallel()

	rig := newPlanRig(t)
	rig.expectMoves([]*dispatchboardports.BoardMove{}, nil)

	plan, err := rig.svc.Plan(t.Context(), &dispatchplanports.Request{TenantInfo: rig.tenant})

	require.NoError(t, err)
	assert.Empty(t, plan.Pairings)
	assert.Empty(t, plan.Uncovered)
	assert.NotNil(t, plan.Pairings, "an empty plan lists nothing rather than null")
	rig.prices.AssertNotCalled(t, "Replace", mock.Anything, mock.Anything)
}

func TestPlan_FailingToRecordPricesFailsThePlan(t *testing.T) {
	t.Parallel()

	rig := newPlanRig(t)
	rig.expectMoves([]*dispatchboardports.BoardMove{rig.move("PRO-1")}, nil)
	rig.expectFleet(newRigDriver("Near", pickupLat+0.1, pickupLon))
	rig.expectStanding()
	boom := errors.New("write failed")
	rig.prices.EXPECT().Replace(mock.Anything, mock.Anything).Return(boom)

	plan, err := rig.svc.Plan(t.Context(), &dispatchplanports.Request{TenantInfo: rig.tenant})

	require.ErrorIs(t, err, boom)
	assert.Nil(t, plan)
}

func TestPlan_FailingToListMovesFailsThePlan(t *testing.T) {
	t.Parallel()

	rig := newPlanRig(t)
	boom := errors.New("read failed")
	rig.expectMoves(nil, boom)

	plan, err := rig.svc.Plan(t.Context(), &dispatchplanports.Request{TenantInfo: rig.tenant})

	require.ErrorIs(t, err, boom)
	assert.Nil(t, plan)
}

func TestPlan_FailingToReadTheLastPlanFailsThePlan(t *testing.T) {
	t.Parallel()

	rig := newPlanRig(t)
	rig.expectMoves([]*dispatchboardports.BoardMove{rig.move("PRO-1")}, nil)
	rig.expectFleet(newRigDriver("Near", pickupLat+0.1, pickupLon))
	boom := errors.New("read failed")
	rig.prices.EXPECT().ListByMoves(mock.Anything, mock.Anything).Return(nil, boom)

	plan, err := rig.svc.Plan(t.Context(), &dispatchplanports.Request{TenantInfo: rig.tenant})

	require.ErrorIs(t, err, boom)
	assert.Nil(t, plan)
}
