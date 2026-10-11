package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func TenderChangeNotices(
	ctx context.Context,
	partners ShipmentPartnerNotices,
	before, after *shipment.Shipment,
	actor *permissionports.RequestActor,
) ([]*agent.RecordChange, error) {
	if partners == nil {
		return nil, nil
	}

	notices, err := partners.PreviewTenderChanges(ctx, before, after, actor)
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(notices))
	for _, notice := range notices {
		changes = append(changes, PartnerNoticeSend(notice, ""))
	}

	return changes, nil
}
