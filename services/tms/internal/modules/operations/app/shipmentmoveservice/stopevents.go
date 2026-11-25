package shipmentmoveservice

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/modules/operations/app/stopactualnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *service) recordStopActualEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	before stopactualnotice.Snapshot,
	origin *shipmentmoveports.StopActualEventOrigin,
) error {
	events := stopactualnotice.Events(tenantInfo, actorForMoveTenant(tenantInfo), move, before)
	for _, params := range events {
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
