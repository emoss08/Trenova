package dispatchplanservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	horizonTestNow = int64(1_700_000_000)
	hour           = int64(3_600)

	denverLat, denverLon   = 39.7392, -104.9903
	chicagoLat, chicagoLon = 41.8781, -87.6298
	omahaLat, omahaLon     = 41.2565, -95.9345
)

type straightLineRoutes struct{}

func (straightLineRoutes) EstimateLegs(
	_ context.Context,
	_ pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	return make([]routeestimateports.LegEstimate, len(legs)), nil
}

type flatMarket struct{}

func (flatMarket) EndMarketCents(float64, float64, int64) int64 { return 0 }

func horizonSnapshot() *dispatchcandidateservice.FleetSnapshot {
	return &dispatchcandidateservice.FleetSnapshot{
		WorkersByID:         map[pulid.ID]*worker.Worker{},
		TractorByID:         map[pulid.ID]*tractor.Tractor{},
		TrailerByID:         map[pulid.ID]*trailer.Trailer{},
		HOSByWorker:         map[pulid.ID]*telematics.WorkerHOSState{},
		PosByTractor:        map[pulid.ID]*telematics.VehiclePosition{},
		CommitmentsByWorker: map[pulid.ID][]*dispatchboardports.WorkerCommitment{},
		NamesByWorker:       map[pulid.ID]string{},
		Now:                 horizonTestNow,
		Model:               dispatchcost.NewModel(new(dispatcheconomics.Default()), nil),
		EndMarket:           flatMarket{},
	}
}

func placedDriver(
	s *dispatchcandidateservice.FleetSnapshot,
	name string,
	lat, lon float64,
) *dispatchboardports.BoardDriver {
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trac_")
	driver := &dispatchboardports.BoardDriver{
		WorkerID:  workerID,
		FirstName: name,
		TractorID: tractorID,
	}
	s.Drivers = append(s.Drivers, driver)
	s.NamesByWorker[workerID] = name
	s.WorkersByID[workerID] = &worker.Worker{
		ID:                   workerID,
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
	}
	s.TractorByID[tractorID] = &tractor.Tractor{
		ID:     tractorID,
		Code:   name + "-T",
		Status: domaintypes.EquipmentStatusAvailable,
	}
	s.PosByTractor[tractorID] = &telematics.VehiclePosition{
		TractorID: tractorID,
		Latitude:  lat,
		Longitude: lon,
	}

	return driver
}

type moveSpec struct {
	fromLat, fromLon float64
	toLat, toLon     float64
	miles            float64
	opens, closes    int64
	due              int64
}

func boardMove(spec moveSpec) *dispatchboardports.BoardMove {
	revenue := 5_000.0
	closes := spec.closes
	return &dispatchboardports.BoardMove{
		MoveID:                 pulid.MustNew("smv_"),
		ProNumber:              "PRO",
		Distance:               &spec.miles,
		Revenue:                &revenue,
		OriginLatitude:         &spec.fromLat,
		OriginLongitude:        &spec.fromLon,
		OriginWindowStart:      spec.opens,
		OriginWindowEnd:        &closes,
		DestinationLatitude:    &spec.toLat,
		DestinationLongitude:   &spec.toLon,
		DestinationWindowStart: spec.due,
	}
}

func TestSuccessorsOf_LinksOnlyReachableFollowOnMoves(t *testing.T) {
	t.Parallel()

	toChicago := boardMove(moveSpec{
		fromLat: denverLat, fromLon: denverLon,
		toLat: chicagoLat, toLon: chicagoLon,
		miles: 1_000,
		opens: horizonTestNow, closes: horizonTestNow + hour, due: horizonTestNow + 30*hour,
	})
	reachable := boardMove(moveSpec{
		fromLat: chicagoLat, fromLon: chicagoLon,
		toLat: omahaLat, toLon: omahaLon,
		miles: 470,
		opens: horizonTestNow + 26*hour, closes: horizonTestNow + 30*hour,
		due: horizonTestNow + 40*hour,
	})
	tooSoon := boardMove(moveSpec{
		fromLat: chicagoLat, fromLon: chicagoLon,
		toLat: omahaLat, toLon: omahaLon,
		miles: 470,
		opens: horizonTestNow + 2*hour, closes: horizonTestNow + 3*hour,
		due: horizonTestNow + 12*hour,
	})

	successors := successorsOf([]*dispatchboardports.BoardMove{toChicago, reachable, tooSoon}, 5)

	assert.Equal(t, []int{1}, successors[0])
}

func TestSuccessorsOf_SkipsMovesWithoutCoordinates(t *testing.T) {
	t.Parallel()

	located := boardMove(moveSpec{
		fromLat: denverLat, fromLon: denverLon,
		toLat: chicagoLat, toLon: chicagoLon,
		miles: 1_000,
	})
	unlocated := &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")}

	successors := successorsOf([]*dispatchboardports.BoardMove{located, unlocated}, 5)

	assert.Empty(t, successors[0])
	assert.Empty(t, successors[1])
}

func TestHorizonBudget_ScalesWithTheBoardWithinBounds(t *testing.T) {
	t.Parallel()

	assert.Equal(t, horizonMinBudget, horizonBudget(10))
	assert.Equal(t, 400*time.Millisecond, horizonBudget(1_000))
	assert.Equal(t, horizonMaxBudget, horizonBudget(5_000))
	assert.Equal(t, horizonMaxBudget, horizonBudget(50_000))
}

func TestIncumbentsOf_MapsStandingProposalsToDrivers(t *testing.T) {
	t.Parallel()

	first := &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")}
	second := &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")}
	moves := []*dispatchboardports.BoardMove{
		{MoveID: pulid.MustNew("smv_")},
		{MoveID: pulid.MustNew("smv_")},
		{MoveID: pulid.MustNew("smv_")},
	}
	graph := &dispatchcandidateservice.Graph{
		Moves:   moves,
		Drivers: []*dispatchboardports.BoardDriver{first, second},
	}

	assert.Nil(t, incumbentsOf(graph, nil))
	assert.Equal(t, []int{1, dispatchplanner.Unassigned, dispatchplanner.Unassigned},
		incumbentsOf(graph, map[pulid.ID]pulid.ID{
			moves[0].MoveID: second.WorkerID,
			moves[2].MoveID: pulid.MustNew("wrk_"),
		}))
}

func TestPlanHorizon_ChainsAFollowOnMoveOntoOneDriver(t *testing.T) {
	t.Parallel()

	snapshot := horizonSnapshot()
	driver := placedDriver(snapshot, "Ada", denverLat, denverLon)
	first := boardMove(moveSpec{
		fromLat: denverLat, fromLon: denverLon,
		toLat: chicagoLat, toLon: chicagoLon,
		miles: 1_000,
		opens: horizonTestNow + hour, closes: horizonTestNow + 2*hour,
		due: horizonTestNow + 30*hour,
	})
	second := boardMove(moveSpec{
		fromLat: chicagoLat, fromLon: chicagoLon,
		toLat: omahaLat, toLon: omahaLon,
		miles: 470,
		opens: horizonTestNow + 26*hour, closes: horizonTestNow + 30*hour,
		due: horizonTestNow + 44*hour,
	})
	candidates := dispatchcandidateservice.New(dispatchcandidateservice.Params{
		Logger:         zap.NewNop(),
		RouteEstimator: straightLineRoutes{},
	})
	graph := candidates.BuildGraph(context.Background(), &dispatchcandidateservice.GraphRequest{
		Moves:    []*dispatchboardports.BoardMove{first, second},
		Snapshot: snapshot,
	})
	plan := &dispatchplanports.Plan{
		Pairings:  []*dispatchplanports.Pairing{},
		Uncovered: []*dispatchplanports.Uncovered{},
		Tours:     []*dispatchplanports.Tour{},
	}

	err := planHorizon(context.Background(), &horizonParams{
		Plan:       plan,
		Graph:      graph,
		Snapshot:   snapshot,
		Candidates: candidates,
		Control: &dispatchcontrol.DispatchControl{
			PlanningMode:             dispatchcontrol.PlanningModeHorizon,
			HorizonMaxMovesPerDriver: 3,
		},
	})

	require.NoError(t, err)
	assert.Empty(t, plan.Uncovered)
	require.Len(t, plan.Tours, 1)
	assert.Equal(t, driver.WorkerID, plan.Tours[0].WorkerID)
	assert.Equal(t, []pulid.ID{first.MoveID, second.MoveID}, plan.Tours[0].MoveIDs)
	require.Len(t, plan.Pairings, 2)
	assert.Equal(t, 0, plan.Pairings[0].SequenceIndex)
	assert.Equal(t, 1, plan.Pairings[1].SequenceIndex)
	assert.Equal(t, plan.Tours[0].TourID, plan.Pairings[1].TourID)
	assert.GreaterOrEqual(
		t,
		plan.Pairings[1].ProjectedStartAt,
		plan.Pairings[0].ProjectedCompleteAt,
	)
	assert.Equal(t, plan.Pairings[0].Candidate.Cost.TotalCents+
		plan.Pairings[1].Candidate.Cost.TotalCents, plan.CostCents)
	assert.Empty(t, snapshot.CommitmentsByWorker[driver.WorkerID],
		"planning leaves the snapshot's commitments as it found them")
}
