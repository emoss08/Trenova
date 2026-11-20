package shipmentmoveservice

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type stopActualTimes struct {
	arrival   *int64
	departure *int64
}

type stopActualSnapshot map[pulid.ID]stopActualTimes

func snapshotStopActuals(move *shipment.ShipmentMove) stopActualSnapshot {
	snapshot := make(stopActualSnapshot, len(move.Stops))
	for _, stop := range move.Stops {
		if stop == nil {
			continue
		}
		snapshot[stop.ID] = stopActualTimes{
			arrival:   cloneActual(stop.ActualArrival),
			departure: cloneActual(stop.ActualDeparture),
		}
	}
	return snapshot
}

func cloneActual(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stopActualEvents(
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	before stopActualSnapshot,
) []*shipmenteventports.RecordShipmentEventParams {
	tenant := tenantRefForMoveTenant(tenantInfo)
	actor := actorForMoveTenant(tenantInfo)
	events := make([]*shipmenteventports.RecordShipmentEventParams, 0)
	for index, stop := range move.ActiveStopsBySequence() {
		previous := before[stop.ID]
		ref := shipmenteventservice.StopActualRef{
			ShipmentID: move.ShipmentID,
			MoveID:     move.ID,
			Stop:       stop,
			Number:     index + 1,
		}
		if kind, ok := actualChange(
			previous.arrival,
			stop.ActualArrival,
			shipmentevent.TypeStopArrived,
			shipmentevent.TypeStopArrivalCorrected,
		); ok {
			events = append(events, shipmenteventservice.BuildStopActual(
				tenant,
				withActual(ref, kind, stop.ActualArrival, previous.arrival),
				actor,
			))
		}
		if kind, ok := actualChange(
			previous.departure,
			stop.ActualDeparture,
			shipmentevent.TypeStopDeparted,
			shipmentevent.TypeStopDepartureCorrected,
		); ok {
			events = append(events, shipmenteventservice.BuildStopActual(
				tenant,
				withActual(ref, kind, stop.ActualDeparture, previous.departure),
				actor,
			))
		}
	}
	return events
}

func actualChange(
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

func (s *service) recordStopActualEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	before stopActualSnapshot,
	origin *shipmentmoveports.StopActualEventOrigin,
) error {
	for _, params := range stopActualEvents(tenantInfo, move, before) {
		if err := s.eventService.Record(ctx, withEventOrigin(params, origin)); err != nil {
			return err
		}
	}
	return nil
}

func withEventOrigin(
	params *shipmenteventports.RecordShipmentEventParams,
	origin *shipmentmoveports.StopActualEventOrigin,
) *shipmenteventports.RecordShipmentEventParams {
	if params == nil || origin == nil {
		return params
	}
	if params.Metadata == nil {
		params.Metadata = make(map[string]any, len(origin.Metadata))
	}
	maps.Copy(params.Metadata, origin.Metadata)
	if origin.ActorLabel != "" {
		params.ActorLabel = origin.ActorLabel
	}
	if origin.CorrelationID != "" {
		params.CorrelationID = origin.CorrelationID
	}
	return params
}
