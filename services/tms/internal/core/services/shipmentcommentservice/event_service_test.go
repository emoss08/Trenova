package shipmentcommentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
)

type noopShipmentEventService struct{}

func (noopShipmentEventService) Record(
	_ context.Context,
	_ *shipmenteventports.RecordShipmentEventParams,
) error {
	return nil
}

func (noopShipmentEventService) List(
	_ context.Context,
	_ *shipmenteventports.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, nil
}
