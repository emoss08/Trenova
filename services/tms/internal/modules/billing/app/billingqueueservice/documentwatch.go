package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const shipmentResourceType = "shipment"

type DocumentObserverParams struct {
	fx.In

	Repo     billingqueueports.BillingQueueRepository
	Realtime realtimeports.RealtimeService `optional:"true"`
	Logger   *zap.Logger
}

type DocumentObserver struct {
	repo     billingqueueports.BillingQueueRepository
	realtime realtimeports.RealtimeService
	l        *zap.Logger
}

var _ documentports.DocumentChangeObserver = (*DocumentObserver)(nil)

func NewDocumentObserver(p DocumentObserverParams) documentports.DocumentChangeObserver {
	return &DocumentObserver{
		repo:     p.Repo,
		realtime: p.Realtime,
		l:        p.Logger.Named("billing-queue.document-observer"),
	}
}

func (o *DocumentObserver) DocumentsChanged(
	ctx context.Context,
	change *documentports.DocumentChange,
) {
	if o.realtime == nil || change == nil || change.ResourceType != shipmentResourceType {
		return
	}
	shipmentIDs := make([]pulid.ID, 0, len(change.ResourceIDs))
	for _, raw := range change.ResourceIDs {
		id, err := pulid.Parse(raw)
		if err != nil || id.IsNil() {
			continue
		}
		shipmentIDs = append(shipmentIDs, id)
	}
	if len(shipmentIDs) == 0 {
		return
	}

	byShipment, err := o.repo.ListActiveInvoiceItemsByShipmentIDs(
		ctx,
		&billingqueueports.ListActiveInvoiceItemsRequest{
			TenantInfo:  change.TenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		o.l.Warn("could not find the billing items behind changed shipment documents",
			zap.Error(err),
		)
		return
	}

	for _, items := range byShipment {
		for _, item := range items {
			if item == nil {
				continue
			}
			if err = realtimeinvalidation.Publish(
				ctx,
				o.realtime,
				&realtimeinvalidation.PublishParams{
					OrganizationID: item.OrganizationID,
					BusinessUnitID: item.BusinessUnitID,
					Resource:       permission.ResourceBillingQueue.String(),
					Action:         string(permission.OpUpdate),
					RecordID:       item.ID,
					Entity: map[string]string{
						"shipmentId": item.ShipmentID.String(),
					},
				},
			); err != nil {
				o.l.Warn("failed to publish billing queue invalidation",
					zap.String("billingQueueItemId", item.ID.String()),
					zap.Error(err),
				)
			}
		}
	}
}
