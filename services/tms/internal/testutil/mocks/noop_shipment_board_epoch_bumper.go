package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ shipmentboardports.ShipmentBoardEpochBumper = (*NoopShipmentBoardEpochBumper)(nil)

type NoopShipmentBoardEpochBumper struct{}

func (n *NoopShipmentBoardEpochBumper) BumpEpoch(_ context.Context, _, _ pulid.ID) error {
	return nil
}
