package customertracking

import "github.com/emoss08/trenova/shared/pulid"

type StopFacts struct {
	ID              pulid.ID
	Pickup          bool
	Delivery        bool
	ActualArrival   *int64
	ActualDeparture *int64
}

type EtaFacts struct {
	StopID           pulid.ID
	NeedsAttention   bool
	Arrived          bool
	EstimatedArrival *int64
	ComputedAt       int64
}

type PodFacts struct {
	DocumentID pulid.ID
	FiledAt    int64
}

// ShipmentFacts is everything Detect needs about one shipment. Stops are in
// travel order: by move sequence, then stop sequence, canceled stops left out.
type ShipmentFacts struct {
	ShipmentID pulid.ID
	CustomerID pulid.ID
	Canceled   bool
	Completed  bool
	Stops      []StopFacts
	Etas       []EtaFacts
	Pod        *PodFacts
}

type Occurrence struct {
	Event       Event
	SubjectKey  string
	StopID      pulid.ID
	OccurredAt  int64
	EstimatedAt *int64
}

// Detect reports the status events the customer is owed for a shipment as it
// stands now. It reads state rather than a stream of changes, so it gives the
// same answer however often, late or out of order it is asked; the ledger's
// unique key turns a repeated answer into one email.
//
// Only events a rule asks for are reported, and only those that happened
// after the rule was switched on and within the lookback, so turning a rule on
// never mails a customer about yesterday's loads.
func Detect(facts *ShipmentFacts, profile *Profile, now int64) []Occurrence {
	if facts == nil || facts.Canceled || !profile.NotifiesAny() ||
		profile.NotificationsEnabledAt == nil {
		return nil
	}
	floor := max(*profile.NotificationsEnabledAt, now-NotificationLookbackHours*3600)
	occurrences := make([]Occurrence, 0, 2)
	keep := func(o Occurrence) {
		if profile.Notifies(o.Event) && o.OccurredAt >= floor && o.OccurredAt <= now {
			occurrences = append(occurrences, o)
		}
	}

	if at, ok := pickedUpAt(facts.Stops); ok {
		keep(Occurrence{Event: EventPickedUp, OccurredAt: at})
	}
	delivered, deliveredTime := deliveredAt(facts)
	if delivered {
		keep(Occurrence{Event: EventDelivered, OccurredAt: deliveredTime})
	}
	if !delivered && !facts.Completed {
		for _, eta := range facts.Etas {
			if !eta.NeedsAttention || eta.Arrived || eta.StopID.IsNil() {
				continue
			}
			keep(Occurrence{
				Event:       EventDelayed,
				SubjectKey:  eta.StopID.String(),
				StopID:      eta.StopID,
				OccurredAt:  eta.ComputedAt,
				EstimatedAt: eta.EstimatedArrival,
			})
		}
	}
	if facts.Pod != nil && !facts.Pod.DocumentID.IsNil() {
		keep(Occurrence{Event: EventPodReady, OccurredAt: facts.Pod.FiledAt})
	}
	return occurrences
}

func pickedUpAt(stops []StopFacts) (int64, bool) {
	var earliest int64
	found := false
	for _, stop := range stops {
		if !stop.Pickup || stop.ActualDeparture == nil {
			continue
		}
		if !found || *stop.ActualDeparture < earliest {
			earliest, found = *stop.ActualDeparture, true
		}
	}
	return earliest, found
}

// deliveredAt is the departure from the last delivery stop. A shipment that
// was closed out without a departure on record counts as delivered at its last
// delivery arrival.
func deliveredAt(facts *ShipmentFacts) (delivered bool, at int64) {
	var last *StopFacts
	for i := range facts.Stops {
		if facts.Stops[i].Delivery {
			last = &facts.Stops[i]
		}
	}
	if last == nil {
		return false, 0
	}
	if last.ActualDeparture != nil {
		return true, *last.ActualDeparture
	}
	if facts.Completed && last.ActualArrival != nil {
		return true, *last.ActualArrival
	}
	return false, 0
}
