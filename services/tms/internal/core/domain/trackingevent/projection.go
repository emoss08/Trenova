package trackingevent

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	reasonCanceled        = "This load has been canceled"
	reasonStopRemoved     = "This stop is no longer on the load"
	reasonEnteredOnRecord = "A time entered on the shipment takes precedence"
	reasonWaitEarlier     = "Waiting for the earlier stops on this load"
	reasonWaitArrival     = "Waiting for the arrival at this stop"
	reasonBeforePrevious  = "Earlier than the previous stop's departure"
	reasonBeforeArrival   = "Earlier than the arrival at this stop"
	reasonAfterDeparture  = "Later than the departure from this stop"
	reasonAfterNext       = "Later than the arrival at the next stop"
	reasonOtherReportUsed = "Another report of this visit was used"
)

type Verdict struct {
	Outcome Outcome
	Reason  string
}

type Projection struct {
	Changes  []shipment.StopActualChange
	Verdicts map[pulid.ID]Verdict
}

func (p Projection) HasChanges() bool {
	return len(p.Changes) > 0
}

type Options struct {
	LockedReason string
}

type bounds struct {
	lower *int64
	upper *int64
}

func (b bounds) admits(at int64) bool {
	return (b.lower == nil || at >= *b.lower) && (b.upper == nil || at <= *b.upper)
}

func (b bounds) refusal(at int64, kind shipment.VisitKind) string {
	if b.lower != nil && at < *b.lower {
		if kind == shipment.VisitArrival {
			return reasonBeforePrevious
		}
		return reasonBeforeArrival
	}
	if kind == shipment.VisitArrival {
		return reasonAfterDeparture
	}
	return reasonAfterNext
}

func Project(move *shipment.ShipmentMove, events []*TrackingEvent, opts Options) Projection {
	projection := Projection{Verdicts: make(map[pulid.ID]Verdict, len(events))}
	stops := move.ActiveStopsBySequence()
	grouped := groupByStop(stops, events, projection.Verdicts)

	frozen := opts.LockedReason
	if move.Status == shipment.MoveStatusCanceled {
		frozen = reasonCanceled
	}

	var previousDeparture *int64
	earlierOpen := false
	for idx, stop := range stops {
		visits := grouped[stop.ID]

		arrivalBounds := bounds{lower: previousDeparture, upper: stop.ActualDeparture}
		arrival := settle(settleInput{
			current:    stop.ActualArrival,
			candidates: visits[shipment.VisitArrival],
			kind:       shipment.VisitArrival,
			frozen:     frozen,
			waitReason: waitReason(earlierOpen, reasonWaitEarlier),
			bounds:     arrivalBounds,
		}, projection.Verdicts)

		var nextArrival *int64
		if idx+1 < len(stops) {
			nextArrival = stops[idx+1].ActualArrival
		}
		departure := settle(settleInput{
			current:    stop.ActualDeparture,
			candidates: visits[shipment.VisitDeparture],
			kind:       shipment.VisitDeparture,
			frozen:     frozen,
			waitReason: waitReason(arrival == nil, reasonWaitArrival),
			bounds:     bounds{lower: arrival, upper: nextArrival},
		}, projection.Verdicts)

		change := shipment.StopActualChange{StopID: stop.ID}
		if differs(stop.ActualArrival, arrival) {
			change.Arrival = arrival
		}
		if differs(stop.ActualDeparture, departure) {
			change.Departure = departure
		}
		if !change.Empty() {
			projection.Changes = append(projection.Changes, change)
		}

		if departure == nil {
			earlierOpen = true
		} else {
			previousDeparture = departure
		}
	}

	return projection
}

func groupByStop(
	stops []*shipment.Stop,
	events []*TrackingEvent,
	verdicts map[pulid.ID]Verdict,
) map[pulid.ID]map[shipment.VisitKind][]*TrackingEvent {
	active := make(map[pulid.ID]struct{}, len(stops))
	for _, stop := range stops {
		active[stop.ID] = struct{}{}
	}

	grouped := make(map[pulid.ID]map[shipment.VisitKind][]*TrackingEvent, len(stops))
	for _, event := range events {
		if event == nil {
			continue
		}
		if _, ok := active[event.StopID]; !ok {
			verdicts[event.ID] = Verdict{Outcome: OutcomeRefused, Reason: reasonStopRemoved}
			continue
		}
		byKind, ok := grouped[event.StopID]
		if !ok {
			byKind = make(map[shipment.VisitKind][]*TrackingEvent, 2)
			grouped[event.StopID] = byKind
		}
		byKind[event.Kind] = append(byKind[event.Kind], event)
	}

	for _, byKind := range grouped {
		for _, candidates := range byKind {
			slices.SortFunc(candidates, compareCandidates)
		}
	}
	return grouped
}

func compareCandidates(a, b *TrackingEvent) int {
	if c := cmp.Compare(b.Source.Precedence(), a.Source.Precedence()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.EventAt, b.EventAt); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Source, b.Source); c != 0 {
		return c
	}
	return cmp.Compare(a.SourceKey, b.SourceKey)
}

type settleInput struct {
	current    *int64
	candidates []*TrackingEvent
	kind       shipment.VisitKind
	frozen     string
	waitReason string
	bounds     bounds
}

func settle(in settleInput, verdicts map[pulid.ID]Verdict) *int64 {
	if len(in.candidates) == 0 {
		return in.current
	}

	if in.frozen != "" {
		markAgainst(in.candidates, in.current, Verdict{
			Outcome: frozenOutcome(in.frozen),
			Reason:  in.frozen,
		}, verdicts)
		return in.current
	}

	if in.current != nil && !anyAt(in.candidates, *in.current) {
		markAll(in.candidates, Verdict{Outcome: OutcomeSuperseded, Reason: reasonEnteredOnRecord}, verdicts)
		return in.current
	}

	if in.waitReason != "" {
		if in.current != nil {
			markAgainst(in.candidates, in.current, Verdict{
				Outcome: OutcomePending,
				Reason:  in.waitReason,
			}, verdicts)
			return in.current
		}
		markAll(in.candidates, Verdict{Outcome: OutcomePending, Reason: in.waitReason}, verdicts)
		return nil
	}

	var chosen *TrackingEvent
	for _, candidate := range in.candidates {
		if in.bounds.admits(candidate.EventAt) {
			chosen = candidate
			break
		}
	}

	if chosen == nil {
		for _, candidate := range in.candidates {
			verdicts[candidate.ID] = Verdict{
				Outcome: OutcomeRefused,
				Reason:  in.bounds.refusal(candidate.EventAt, in.kind),
			}
		}
		return in.current
	}

	for _, candidate := range in.candidates {
		switch {
		case candidate == chosen:
			verdicts[candidate.ID] = Verdict{Outcome: OutcomeApplied}
		case candidate.EventAt == chosen.EventAt:
			verdicts[candidate.ID] = Verdict{Outcome: OutcomeDuplicate}
		default:
			verdicts[candidate.ID] = Verdict{Outcome: OutcomeSuperseded, Reason: reasonOtherReportUsed}
		}
	}
	at := chosen.EventAt
	return &at
}

func markAgainst(
	candidates []*TrackingEvent,
	current *int64,
	otherwise Verdict,
	verdicts map[pulid.ID]Verdict,
) {
	appliedSet := false
	for _, candidate := range candidates {
		if current != nil && candidate.EventAt == *current {
			if !appliedSet {
				verdicts[candidate.ID] = Verdict{Outcome: OutcomeApplied}
				appliedSet = true
				continue
			}
			verdicts[candidate.ID] = Verdict{Outcome: OutcomeDuplicate}
			continue
		}
		verdicts[candidate.ID] = otherwise
	}
}

func markAll(candidates []*TrackingEvent, verdict Verdict, verdicts map[pulid.ID]Verdict) {
	for _, candidate := range candidates {
		verdicts[candidate.ID] = verdict
	}
}

func anyAt(candidates []*TrackingEvent, at int64) bool {
	for _, candidate := range candidates {
		if candidate.EventAt == at {
			return true
		}
	}
	return false
}

func frozenOutcome(reason string) Outcome {
	if reason == reasonCanceled {
		return OutcomeRefused
	}
	return OutcomeSuperseded
}

func waitReason(waiting bool, reason string) string {
	if waiting {
		return reason
	}
	return ""
}

func differs(current, target *int64) bool {
	if target == nil {
		return false
	}
	return current == nil || *current != *target
}
