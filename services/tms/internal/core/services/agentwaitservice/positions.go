package agentwaitservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// positionFreshFor is how old a polled position may be and still say where a
// truck is now.
const positionFreshFor int64 = 15 * 60

// watchedMove is a move an arrival or departure wait watches, with the
// tractor whose position says where it is.
type watchedMove struct {
	move    *shipment.ShipmentMove
	tractor pulid.ID
}

// NotifyPositions checks the polled positions against the stops that open
// arrival and departure waits watch. It ends a wait the provider's geofence
// events and the recorded stop actuals never reach: a provider that sends no
// geofence events, a geofence the provider does not hold, or an organization
// that records stop actuals by hand. A move and its tractor are read once per
// poll, however many waits watch it.
func (s *Service) NotifyPositions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	positions []*telematics.VehiclePosition,
) {
	if len(positions) == 0 {
		return
	}
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), tenant.DBTenant())

	waits, err := s.waits.ListOpenOfKinds(ctx, &repositories.ListOpenWaitsOfKindsRequest{
		TenantInfo: tenant,
		Kinds:      []agentwait.Kind{agentwait.KindStopArrival, agentwait.KindStopDeparture},
	})
	if err != nil {
		s.l.Warn("could not read the waits on arrivals and departures", zap.Error(err))
		return
	}
	if len(waits) == 0 {
		return
	}

	now := s.now()
	byTractor := make(map[pulid.ID]*telematics.VehiclePosition, len(positions))
	for _, position := range positions {
		if position == nil || now-position.RecordedAt > positionFreshFor {
			continue
		}
		if known := byTractor[position.TractorID]; known == nil ||
			position.RecordedAt > known.RecordedAt {
			byTractor[position.TractorID] = position
		}
	}
	if len(byTractor) == 0 {
		return
	}

	moves := make(map[pulid.ID]*watchedMove, len(waits))
	for _, wait := range waits {
		watched := s.watched(ctx, tenant, wait.Condition.ShipmentMoveID, moves)
		if watched == nil {
			continue
		}
		position := byTractor[watched.tractor]
		if position == nil {
			continue
		}
		stop := visitStop(watched.move, wait)
		if stop == nil || stop.Location == nil {
			continue
		}
		if err = stop.Location.PopulateGeofenceVertices(); err != nil {
			continue
		}
		inside := stop.Location.GeofenceContains(position.Latitude, position.Longitude)
		s.observeVisit(ctx, wait, stop, position, inside)
	}
}

// watched reads a move and the tractor assigned to it, once per poll. A move
// with no tractor assigned has no position to read and is remembered as nil.
func (s *Service) watched(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID pulid.ID,
	cache map[pulid.ID]*watchedMove,
) *watchedMove {
	if watched, seen := cache[moveID]; seen {
		return watched
	}
	cache[moveID] = nil

	move, err := s.facts.move(ctx, tenant, moveID)
	if err != nil || s.facts.assignments == nil {
		return nil
	}
	assignment, err := s.facts.assignments.GetByMoveID(ctx, tenant, moveID)
	if err != nil || assignment == nil || assignment.TractorID == nil ||
		assignment.TractorID.IsNil() {
		return nil
	}

	watched := &watchedMove{move: move, tractor: *assignment.TractorID}
	cache[moveID] = watched

	return watched
}

// visitStop is the stop a visit wait watches: the one it names, or else the
// move's next arrival, or for a departure the stop it is at.
func visitStop(move *shipment.ShipmentMove, wait *agentwait.Wait) *shipment.Stop {
	if wait.Condition.StopID.IsNotNil() {
		return stopOf(move, wait.Condition.StopID)
	}

	stops := move.ActiveStopsBySequence()
	if wait.Kind == agentwait.KindStopDeparture {
		for _, stop := range stops {
			if stop.ActualArrival != nil && stop.ActualDeparture == nil {
				return stop
			}
		}
	}
	for _, stop := range stops {
		if stop.ActualArrival == nil {
			return stop
		}
	}

	return nil
}

// observeVisit ends an arrival wait on a truck inside the stop's geofence. A
// departure is being outside after being inside: the first sight inside is
// kept on the wait, so a truck that left between two polls is still seen to
// have left, and one already recorded as arrived needs no sight.
func (s *Service) observeVisit(
	ctx context.Context,
	wait *agentwait.Wait,
	stop *shipment.Stop,
	position *telematics.VehiclePosition,
	inside bool,
) {
	switch {
	case wait.Kind == agentwait.KindStopArrival && inside:
		s.meet(ctx, wait, fmt.Sprintf("The truck is inside the geofence of %s, as of %s.",
			stopName(stop), stamp(position.RecordedAt)))
	case wait.Kind == agentwait.KindStopDeparture && inside:
		if wait.Condition.SeenInsideAt != 0 {
			return
		}
		condition := *wait.Condition
		condition.SeenInsideAt = position.RecordedAt
		if err := s.waits.UpdateCondition(ctx, &repositories.UpdateAgentWaitConditionRequest{
			ID:         wait.ID,
			TenantInfo: tenantOf(wait),
			Condition:  &condition,
		}); err != nil {
			s.l.Warn("could not keep that a truck was seen at its stop",
				zap.String("wait", wait.ID.String()), zap.Error(err))
			return
		}
		wait.Condition = &condition
	case wait.Kind == agentwait.KindStopDeparture &&
		(wait.Condition.SeenInsideAt != 0 || stop.ActualArrival != nil):
		s.meet(ctx, wait, fmt.Sprintf("The truck has left the geofence of %s; seen outside "+
			"it at %s.", stopName(stop), stamp(position.RecordedAt)))
	}
}
