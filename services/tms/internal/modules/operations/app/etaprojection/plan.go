package etaprojection

import (
	"sort"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	PositionStaleAfterSeconds  = int64(3600)
	CheckCallStaleAfterSeconds = int64(4 * 3600)
	HOSStaleAfterSeconds       = int64(12 * 3600)
	reportedEtaGraceSeconds    = int64(15 * 60)
	secondsPerMinute           = int64(60)
	msPerSecond                = int64(1000)
	msPerMinute                = int64(60_000)
)

type Settings struct {
	ThresholdMinutes int64
	DwellMinutes     int64
}

type Input struct {
	Now         int64
	Shipment    *shipment.Shipment
	Assignments map[pulid.ID]*shipmentports.BoardMove
	Positions   map[pulid.ID]*telematics.VehiclePosition
	HOS         map[pulid.ID]*telematics.WorkerHOSState
	HOSExempt   map[pulid.ID]bool
	CheckCalls  map[pulid.ID]*checkcall.CheckCall
	Settings    Settings
}

type plannedStop struct {
	move  *shipment.ShipmentMove
	stop  *shipment.Stop
	point *routeestimateports.Point
	leg   int
}

type origin struct {
	basis           stopeta.Basis
	point           *routeestimateports.Point
	at              int64
	inputAt         int64
	stale           bool
	reportedArrival int64
	atStop          bool
}

type Plan struct {
	in     *Input
	stops  []plannedStop
	origin origin
	legs   []routeestimateports.Leg
}

func Prepare(in *Input) *Plan {
	plan := &Plan{in: in}
	if in == nil || in.Shipment == nil || !stopeta.TrackedShipment(in.Shipment.Status) {
		return plan
	}

	ordered := orderedStops(in.Shipment)
	var lastDeparted *plannedStop
	for i := range ordered {
		item := ordered[i]
		if openStop(item.move, item.stop) {
			plan.stops = append(plan.stops, item)
			continue
		}
		if len(plan.stops) == 0 && item.stop.ActualDeparture != nil && !item.stop.IsCanceled() {
			lastDeparted = &ordered[i]
		}
	}
	if len(plan.stops) == 0 {
		return plan
	}

	plan.origin = plan.chooseOrigin(lastDeparted)
	plan.buildLegs()

	return plan
}

func (p *Plan) Legs() []routeestimateports.Leg {
	return p.legs
}

func (p *Plan) chooseOrigin(lastDeparted *plannedStop) origin {
	first := p.stops[0]
	now := p.in.Now

	if first.stop.ActualArrival != nil {
		return origin{
			basis:   stopeta.BasisAtStop,
			point:   first.point,
			at:      now,
			inputAt: *first.stop.ActualArrival,
			atStop:  true,
		}
	}

	board := p.in.Assignments[first.move.ID]
	position := p.positionFor(board)
	call := p.in.CheckCalls[first.move.ID]

	if position != nil && now-position.RecordedAt <= PositionStaleAfterSeconds {
		return positionOrigin(position, now, false)
	}
	if arrival, ok := reportedArrival(call, first.stop.ID, now); ok {
		return origin{
			basis:           stopeta.BasisCarrierReported,
			at:              now,
			inputAt:         call.ReportedAt,
			reportedArrival: arrival,
		}
	}
	if point := checkCallPoint(call); point != nil &&
		(position == nil || call.ReportedAt > position.RecordedAt) {
		return origin{
			basis:   stopeta.BasisCheckCall,
			point:   point,
			at:      now,
			inputAt: call.ReportedAt,
			stale:   now-call.ReportedAt > CheckCallStaleAfterSeconds,
		}
	}
	if position != nil {
		return positionOrigin(position, now, true)
	}
	if lastDeparted != nil && lastDeparted.point != nil {
		departed := *lastDeparted.stop.ActualDeparture
		return origin{
			basis:   stopeta.BasisLastDeparture,
			point:   lastDeparted.point,
			at:      departed,
			inputAt: departed,
		}
	}

	return origin{basis: stopeta.BasisNone}
}

func (p *Plan) positionFor(board *shipmentports.BoardMove) *telematics.VehiclePosition {
	if board == nil || board.AssignedTractorID.IsNil() {
		return nil
	}
	position := p.in.Positions[board.AssignedTractorID]
	if position == nil || !validPoint(position.Latitude, position.Longitude) {
		return nil
	}

	return position
}

func (p *Plan) buildLegs() {
	previous := p.origin.point
	chained := p.origin.basis != stopeta.BasisNone
	for i := range p.stops {
		item := &p.stops[i]
		item.leg = -1
		if !chained {
			continue
		}
		if i == 0 && (p.origin.atStop || p.origin.basis == stopeta.BasisCarrierReported) {
			previous = item.point
			if item.point == nil {
				chained = false
			}
			continue
		}
		if previous == nil || item.point == nil {
			chained = false
			continue
		}
		item.leg = len(p.legs)
		p.legs = append(p.legs, routeestimateports.Leg{From: *previous, To: *item.point})
		previous = item.point
	}
}

func positionOrigin(position *telematics.VehiclePosition, now int64, stale bool) origin {
	return origin{
		basis: stopeta.BasisPosition,
		point: &routeestimateports.Point{
			Latitude:  position.Latitude,
			Longitude: position.Longitude,
		},
		at:      now,
		inputAt: position.RecordedAt,
		stale:   stale,
	}
}

func reportedArrival(call *checkcall.CheckCall, stopID pulid.ID, now int64) (int64, bool) {
	if call == nil || call.EstimatedArrival == nil {
		return 0, false
	}
	if call.StopID.IsNotNil() && call.StopID != stopID {
		return 0, false
	}
	if now-call.ReportedAt > CheckCallStaleAfterSeconds {
		return 0, false
	}
	if *call.EstimatedArrival < now-reportedEtaGraceSeconds {
		return 0, false
	}

	return max(*call.EstimatedArrival, now), true
}

func checkCallPoint(call *checkcall.CheckCall) *routeestimateports.Point {
	if call == nil || call.Latitude == nil || call.Longitude == nil ||
		!validPoint(*call.Latitude, *call.Longitude) {
		return nil
	}

	return &routeestimateports.Point{Latitude: *call.Latitude, Longitude: *call.Longitude}
}

func orderedStops(sp *shipment.Shipment) []plannedStop {
	moves := make([]*shipment.ShipmentMove, 0, len(sp.Moves))
	for _, move := range sp.Moves {
		if move != nil && move.Status != shipment.MoveStatusCanceled {
			moves = append(moves, move)
		}
	}
	sort.SliceStable(moves, func(i, j int) bool { return moves[i].Sequence < moves[j].Sequence })

	out := make([]plannedStop, 0, len(moves)*2)
	for _, move := range moves {
		stops := make([]*shipment.Stop, 0, len(move.Stops))
		for _, stop := range move.Stops {
			if stop != nil {
				stops = append(stops, stop)
			}
		}
		sort.SliceStable(
			stops,
			func(i, j int) bool { return stops[i].Sequence < stops[j].Sequence },
		)
		for _, stop := range stops {
			out = append(out, plannedStop{move: move, stop: stop, point: stopPoint(stop), leg: -1})
		}
	}

	return out
}

func openStop(move *shipment.ShipmentMove, stop *shipment.Stop) bool {
	if move.Status == shipment.MoveStatusCompleted || move.Status == shipment.MoveStatusCanceled {
		return false
	}

	return !stop.IsCompleted() && !stop.IsCanceled() && stop.ActualDeparture == nil
}

func stopPoint(stop *shipment.Stop) *routeestimateports.Point {
	if stop.Location == nil || stop.Location.Latitude == nil || stop.Location.Longitude == nil {
		return nil
	}
	if !validPoint(*stop.Location.Latitude, *stop.Location.Longitude) {
		return nil
	}

	return &routeestimateports.Point{
		Latitude:  *stop.Location.Latitude,
		Longitude: *stop.Location.Longitude,
	}
}

func validPoint(lat, lon float64) bool {
	if lat == 0 && lon == 0 {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
