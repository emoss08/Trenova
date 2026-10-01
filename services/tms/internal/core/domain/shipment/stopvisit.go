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
	slices.SortStableFunc(stops, func(a, b *Stop) int { return cmp.Compare(a.Sequence, b.Sequence) })

	return stops
}

func (m *ShipmentMove) MatchObservedVisit(
	locationID pulid.ID,
	kind VisitKind,
) (*Stop, VisitMatch) {
	if locationID.IsNil() {
		return nil, VisitMatchNoStop
	}

	var atLocation, onSite, unarrived, undeparted *Stop
	for _, stop := range m.ActiveStopsBySequence() {
		if stop.LocationID != locationID {
			continue
		}
		if atLocation == nil {
			atLocation = stop
		}
		switch {
		case stop.ActualArrival != nil && stop.ActualDeparture == nil:
			if onSite == nil {
				onSite = stop
			}
		case stop.ActualArrival == nil && stop.ActualDeparture == nil:
			if unarrived == nil {
				unarrived = stop
			}
		}
		if stop.ActualDeparture == nil && undeparted == nil {
			undeparted = stop
		}
	}

	if atLocation == nil {
		return nil, VisitMatchNoStop
	}

	switch kind {
	case VisitArrival:
		if onSite != nil || unarrived == nil {
			return nil, VisitMatchDuplicate
		}
		return unarrived, VisitMatchStop
	case VisitDeparture:
		if onSite != nil {
			return onSite, VisitMatchStop
		}
		if undeparted != nil {
			return undeparted, VisitMatchStop
		}
		return nil, VisitMatchDuplicate
	default:
		return nil, VisitMatchNoStop
	}
}
