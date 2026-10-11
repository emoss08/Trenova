package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func QueueRecord(item *billingqueue.BillingQueueItem) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceBillingQueue,
		ID:       item.ID,
		Label:    item.Number,
		Version:  PreviewVersion(item.Version),
	}
}

func ShipmentLabel(item *billingqueue.BillingQueueItem) string {
	if item.Shipment != nil && item.Shipment.ProNumber != "" {
		return item.Shipment.ProNumber
	}

	return LabelShipment
}
