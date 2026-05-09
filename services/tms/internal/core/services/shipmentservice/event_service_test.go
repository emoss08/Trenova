package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

type noopShipmentEventService struct{}

func (noopShipmentEventService) Record(
	_ context.Context,
	_ *services.RecordShipmentEventParams,
) error {
	return nil
}

func (noopShipmentEventService) List(
	_ context.Context,
	_ *repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, nil
}
