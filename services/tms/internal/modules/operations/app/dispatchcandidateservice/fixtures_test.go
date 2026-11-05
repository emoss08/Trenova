package dispatchcandidateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	testNow    = int64(1_700_000_000)
	testHourMs = int64(3_600_000)

	denverLat, denverLon   = 39.7392, -104.9903
	chicagoLat, chicagoLon = 41.8781, -87.6298
)

type fakeRoutes struct {
	miles   float64
	seconds int64
	routed  bool
}

func (f fakeRoutes) EstimateLegs(
	_ context.Context,
	_ pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	out := make([]routeestimateports.LegEstimate, len(legs))
	for i := range out {
		out[i] = routeestimateports.LegEstimate{
			Miles:        f.miles,
			DriveSeconds: f.seconds,
			Routed:       f.routed,
		}
	}

	return out, nil
}

func newTestService(routes routeestimateports.RouteEstimator) *Service {
	return &Service{l: zap.NewNop(), routes: routes}
}

func newTestSnapshot() *FleetSnapshot {
	return &FleetSnapshot{
		WorkersByID:         map[pulid.ID]*worker.Worker{},
		TractorByID:         map[pulid.ID]*tractor.Tractor{},
		TrailerByID:         map[pulid.ID]*trailer.Trailer{},
		HOSByWorker:         map[pulid.ID]*telematics.WorkerHOSState{},
		PosByTractor:        map[pulid.ID]*telematics.VehiclePosition{},
		CommitmentsByWorker: map[pulid.ID][]*dispatchboardports.WorkerCommitment{},
		AcceptanceByWorker:  map[pulid.ID]*dispatchboardports.WorkerAcceptanceStats{},
		NamesByWorker:       map[pulid.ID]string{},
		Now:                 testNow,
		Model:               dispatchcost.NewModel(new(dispatcheconomics.Default()), nil),
		EndMarket:           noEndMarket{},
	}
}

func withCandidatesPerMove(s *FleetSnapshot, count int) {
	economics := dispatcheconomics.Default()
	economics.CandidatesPerMove = count
	s.Model = dispatchcost.NewModel(&economics, nil)
}

func addDriver(s *FleetSnapshot, name string) *dispatchboardports.BoardDriver {
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

	return driver
}

func placeDriver(s *FleetSnapshot, driver *dispatchboardports.BoardDriver, lat, lon float64) {
	s.PosByTractor[driver.TractorID] = &telematics.VehiclePosition{
		TractorID: driver.TractorID,
		Latitude:  lat,
		Longitude: lon,
	}
}

func addPlacedDriver(
	s *FleetSnapshot,
	name string,
	lat, lon float64,
) *dispatchboardports.BoardDriver {
	driver := addDriver(s, name)
	placeDriver(s, driver, lat, lon)

	return driver
}

func pickupAt(lat, lon, loadedMiles float64) *dispatchboardports.BoardMove {
	return &dispatchboardports.BoardMove{
		MoveID:          pulid.MustNew("smv_"),
		ProNumber:       "PRO-1",
		Distance:        &loadedMiles,
		OriginLatitude:  &lat,
		OriginLongitude: &lon,
	}
}

func evaluateFor(
	s *FleetSnapshot,
	driver *dispatchboardports.BoardDriver,
	move *dispatchboardports.BoardMove,
	routed *routedLeg,
) *evaluation {
	s.Prepare()

	return evaluate(s, s.profile(driver), move, routed)
}

func costLine(cost dispatcheconomics.Cost, component dispatcheconomics.Component) int64 {
	for _, line := range cost.Lines {
		if line.Component == component {
			return line.Cents
		}
	}

	return 0
}

func findingCodes(eval *evaluation) []string {
	codes := make([]string, 0, len(eval.findings))
	for i := range eval.findings {
		codes = append(codes, eval.findings[i].Code)
	}

	return codes
}
