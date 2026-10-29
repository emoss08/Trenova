package stopvisit

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

type Match string

const (
	MatchStop      = Match("Stop")
	MatchDuplicate = Match("Duplicate")
	MatchNoStop    = Match("NoStop")
)

type visitCandidates struct {
	atLocation   *shipment.Stop
	onSite       *shipment.Stop
	unarrived    *shipment.Stop
	undeparted   *shipment.Stop
	lastArrived  *shipment.Stop
	lastDeparted *shipment.Stop
}

func (c *visitCandidates) consider(stop *shipment.Stop) {
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
	if arrived {
		c.lastArrived = stop
	}
	if departed {
		c.lastDeparted = stop
	}
}

func (c *visitCandidates) reported(kind shipmentvocab.VisitKind) *shipment.Stop {
	switch kind {
	case shipmentvocab.VisitArrival:
		return firstStop(c.onSite, c.unarrived, c.lastArrived)
	case shipmentvocab.VisitDeparture:
		return firstStop(c.onSite, c.undeparted, c.lastDeparted)
	default:
		return nil
	}
}

func firstStop(stops ...*shipment.Stop) *shipment.Stop {
	for _, stop := range stops {
		if stop != nil {
			return stop
		}
	}
	return nil
}

func (c *visitCandidates) match(kind shipmentvocab.VisitKind) (*shipment.Stop, Match) {
	if c.atLocation == nil {
		return nil, MatchNoStop
	}

	switch kind {
	case shipmentvocab.VisitArrival:
		if c.onSite != nil || c.unarrived == nil {
			return nil, MatchDuplicate
		}
		return c.unarrived, MatchStop
	case shipmentvocab.VisitDeparture:
		if c.onSite != nil {
			return c.onSite, MatchStop
		}
		if c.undeparted != nil {
			return c.undeparted, MatchStop
		}
		return nil, MatchDuplicate
	default:
		return nil, MatchNoStop
	}
}

func MatchObserved(
	m *shipment.ShipmentMove,
	locationID pulid.ID,
	kind shipmentvocab.VisitKind,
) (*shipment.Stop, Match) {
	return candidatesAt(m, locationID).match(kind)
}

func StopForObserved(
	m *shipment.ShipmentMove,
	locationID pulid.ID,
	kind shipmentvocab.VisitKind,
) *shipment.Stop {
	return candidatesAt(m, locationID).reported(kind)
}

func StopForReported(
	m *shipment.ShipmentMove,
	matches func(*shipment.Stop) bool,
	kind shipmentvocab.VisitKind,
) *shipment.Stop {
	return candidatesWhere(m, matches).reported(kind)
}

func candidatesAt(m *shipment.ShipmentMove, locationID pulid.ID) *visitCandidates {
	if locationID.IsNil() {
		return &visitCandidates{}
	}
	return candidatesWhere(
		m,
		func(stop *shipment.Stop) bool { return stop.LocationID == locationID },
	)
}

func candidatesWhere(m *shipment.ShipmentMove, matches func(*shipment.Stop) bool) *visitCandidates {
	candidates := &visitCandidates{}
	for _, stop := range m.ActiveStopsBySequence() {
		if matches(stop) {
			candidates.consider(stop)
		}
	}
	return candidates
}
