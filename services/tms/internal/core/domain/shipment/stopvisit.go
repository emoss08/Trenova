package shipment

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

type VisitKind string

const (
	VisitArrival   = VisitKind("Arrival")
	VisitDeparture = VisitKind("Departure")
)

type VisitMatch string

const (
	VisitMatchStop      = VisitMatch("Stop")
	VisitMatchDuplicate = VisitMatch("Duplicate")
	VisitMatchNoStop    = VisitMatch("NoStop")
)

func (m *ShipmentMove) ActiveStopsBySequence() []*Stop {
	stops := make([]*Stop, 0, len(m.Stops))
	for _, stop := range m.Stops {
		if stop != nil && stop.Status != StopStatusCanceled {
			stops = append(stops, stop)
		}
	}
	slices.SortStableFunc(stops, func(a, b *Stop) int {
		return cmp.Compare(a.Sequence, b.Sequence)
	})

	return stops
}

type visitCandidates struct {
	atLocation *Stop
	onSite     *Stop
	unarrived  *Stop
	undeparted *Stop
}

func (c *visitCandidates) consider(stop *Stop) {
	if c.atLocation == nil {
		c.atLocation = stop
	}
	arrived := stop.ActualArrival != nil
	departed := stop.ActualDeparture != nil
	if arrived && !departed && c.onSite == nil {
		c.onSite = stop
	}
	if !arrived && !departed && c.unarrived == nil {
		c.unarrived = stop
	}
	if !departed && c.undeparted == nil {
		c.undeparted = stop
	}
}

func (c *visitCandidates) match(kind VisitKind) (*Stop, VisitMatch) {
	if c.atLocation == nil {
		return nil, VisitMatchNoStop
	}

	switch kind {
	case VisitArrival:
		if c.onSite != nil || c.unarrived == nil {
			return nil, VisitMatchDuplicate
		}
		return c.unarrived, VisitMatchStop
	case VisitDeparture:
		if c.onSite != nil {
			return c.onSite, VisitMatchStop
		}
		if c.undeparted != nil {
			return c.undeparted, VisitMatchStop
		}
		return nil, VisitMatchDuplicate
	default:
		return nil, VisitMatchNoStop
	}
}

func (m *ShipmentMove) MatchObservedVisit(
	locationID pulid.ID,
	kind VisitKind,
) (*Stop, VisitMatch) {
	if locationID.IsNil() {
		return nil, VisitMatchNoStop
	}

	var candidates visitCandidates
	for _, stop := range m.ActiveStopsBySequence() {
		if stop.LocationID == locationID {
			candidates.consider(stop)
		}
	}

	return candidates.match(kind)
}
