// Package stopactualnotice says what a change to a move's stop times means:
// the shipment events it records and the arrivals and departures it tells
// agents of. Every path that writes stop times, a recorded actual, a
// reconciled feed or a saved shipment, compares the move before and after
// through here, so none of them leaves a waiting agent or an EDI partner
// unaware that a truck arrived.
package stopactualnotice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type times struct {
	arrival   *int64
	departure *int64
}

// Snapshot is a move's stop times at one moment, by stop.
type Snapshot map[pulid.ID]times

// Take is the move's stop times as they stand.
func Take(move *shipment.ShipmentMove) Snapshot {
	snapshot := make(Snapshot, len(move.Stops))
	for _, stop := range move.Stops {
		if stop == nil {
			continue
		}
		snapshot[stop.ID] = times{
			arrival:   clone(stop.ActualArrival),
			departure: clone(stop.ActualDeparture),
		}
	}

	return snapshot
}

// TakeAll is the stop times of every move, by move.
func TakeAll(moves []*shipment.ShipmentMove) map[pulid.ID]Snapshot {
	snapshots := make(map[pulid.ID]Snapshot, len(moves))
	for _, move := range moves {
		if move == nil || move.ID.IsNil() {
			continue
		}
		snapshots[move.ID] = Take(move)
	}

	return snapshots
}

func clone(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value

	return &cloned
}

// Events are the shipment events for what changed on the move's stops since
// before: an arrival or departure first recorded, or one corrected.
func Events(
	tenantInfo pagination.TenantInfo,
	actor kernelports.AuditActor,
	move *shipment.ShipmentMove,
	before Snapshot,
) []*shipmenteventports.RecordShipmentEventParams {
	tenant := shipmenteventservice.TenantRef{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	events := make([]*shipmenteventports.RecordShipmentEventParams, 0)
	for index, stop := range move.ActiveStopsBySequence() {
		previous := before[stop.ID]
		ref := shipmenteventservice.StopActualRef{
			ShipmentID: move.ShipmentID,
			MoveID:     move.ID,
			Stop:       stop,
			Number:     index + 1,
		}
		if kind, ok := change(
			previous.arrival,
			stop.ActualArrival,
			shipmentevent.TypeStopArrived,
			shipmentevent.TypeStopArrivalCorrected,
		); ok {
			events = append(events, shipmenteventservice.BuildStopActual(
				tenant, withActual(ref, kind, stop.ActualArrival, previous.arrival), actor))
		}
		if kind, ok := change(
			previous.departure,
			stop.ActualDeparture,
			shipmentevent.TypeStopDeparted,
			shipmentevent.TypeStopDepartureCorrected,
		); ok {
			events = append(events, shipmenteventservice.BuildStopActual(
				tenant, withActual(ref, kind, stop.ActualDeparture, previous.departure), actor))
		}
	}

	return events
}

func change(
	before, after *int64,
	recorded, corrected shipmentevent.Type,
) (shipmentevent.Type, bool) {
	switch {
	case after == nil:
		return "", false
	case before == nil:
		return recorded, true
	case *before != *after:
		return corrected, true
	default:
		return "", false
	}
}

func withActual(
	ref shipmenteventservice.StopActualRef,
	kind shipmentevent.Type,
	actual, previous *int64,
) shipmenteventservice.StopActualRef {
	ref.Type = kind
	ref.ActualAt = *actual
	if kind == shipmentevent.TypeStopArrivalCorrected ||
		kind == shipmentevent.TypeStopDepartureCorrected {
		ref.PreviousActualAt = previous
	}

	return ref
}

// Visit is a truck reaching or leaving a stop.
type Visit struct {
	MoveID   pulid.ID
	StopID   pulid.ID
	Departed bool
	At       int64
}

// Visits are the arrivals and departures first recorded on the move since
// before, in stop order. A corrected time is not a new visit: the truck was
// already known to be there.
func Visits(move *shipment.ShipmentMove, before Snapshot) []Visit {
	visits := make([]Visit, 0)
	for _, stop := range move.ActiveStopsBySequence() {
		previous := before[stop.ID]
		if previous.arrival == nil && stop.ActualArrival != nil {
			visits = append(visits, Visit{
				MoveID: move.ID, StopID: stop.ID, At: *stop.ActualArrival,
			})
		}
		if previous.departure == nil && stop.ActualDeparture != nil {
			visits = append(visits, Visit{
				MoveID: move.ID, StopID: stop.ID, Departed: true, At: *stop.ActualDeparture,
			})
		}
	}

	return visits
}

// Publish wakes the agents and waits that watch the move for this visit. The
// event carries only ids: an agent that wants the stop reads it back.
func Publish(
	ctx context.Context,
	publisher agenteventports.AgentEventPublisher,
	tenantInfo pagination.TenantInfo,
	visit Visit,
) {
	kind, verb := agenttrigger.EventShipmentMoveArrived, "Arrived at"
	if visit.Departed {
		kind, verb = agenttrigger.EventShipmentMoveDeparted, "Left"
	}

	agenteventports.PublishAgentEvent(ctx, publisher, agenteventports.AgentEvent{
		Kind:       kind,
		SubjectID:  visit.MoveID,
		TenantInfo: tenantInfo,
		Related:    []pulid.ID{visit.StopID},
		Detail: fmt.Sprintf("%s stop %s at %s.", verb, visit.StopID,
			time.Unix(visit.At, 0).UTC().Format("Mon Jan 2 15:04 MST")),
	})
}
