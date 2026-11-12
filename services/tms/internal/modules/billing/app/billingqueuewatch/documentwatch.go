package billingqueuewatch

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
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
	Rechecks billingqueueports.RecheckRequester
	Realtime realtimeports.RealtimeService
	Logger   *zap.Logger
}

type DocumentObserver struct {
	repo     billingqueueports.BillingQueueRepository
	rechecks billingqueueports.RecheckRequester
	realtime realtimeports.RealtimeService
	l        *zap.Logger
}

var _ documentports.DocumentChangeObserver = (*DocumentObserver)(nil)

func NewDocumentObserver(p DocumentObserverParams) documentports.DocumentChangeObserver {
	return &DocumentObserver{
		repo:     p.Repo,
		rechecks: p.Rechecks,
		realtime: p.Realtime,
		l:        p.Logger.Named("billing-queue.document-observer"),
	}
}

func (o *DocumentObserver) DocumentsChanged(
	ctx context.Context,
	change *documentports.DocumentChange,
) {
	if change == nil || change.ResourceType != shipmentResourceType {
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

	retired := make(map[pulid.ID]pulid.ID, len(change.Retired))
	for _, gone := range change.Retired {
		retired[gone.ID] = gone.ReplacedBy
	}

	itemIDs := make([]pulid.ID, 0, len(byShipment))
	for _, items := range byShipment {
		for _, item := range items {
			if item == nil {
				continue
			}
			o.retirePacketDocuments(ctx, change, item, retired)
			itemIDs = append(itemIDs, item.ID)

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

	if err = o.rechecks.RequestRecheck(ctx, change.TenantInfo, itemIDs...); err != nil {
		o.l.Warn("could not mark the billing items behind changed shipment documents for their checks",
			zap.Error(err),
		)
	}
}

func (o *DocumentObserver) retirePacketDocuments(
	ctx context.Context,
	change *documentports.DocumentChange,
	item *billingqueue.BillingQueueItem,
	retired map[pulid.ID]pulid.ID,
) {
	packet, rotations, changed := withoutRetired(item, retired)
	if !changed {
		return
	}
	kept, err := o.repo.SetPacketDocuments(ctx, &billingqueueports.SetPacketDocumentsRequest{
		TenantInfo: change.TenantInfo,
		ItemID:     item.ID,
		Version:    item.Version,
		Packet:     packet,
		Rotations:  rotations,
	})
	if err != nil {
		o.l.Warn("could not take retired documents out of an invoice packet",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.Error(err),
		)
		return
	}
	if !kept {
		o.l.Debug("an invoice packet changed while its retired documents were taken out",
			zap.String("billingQueueItemId", item.ID.String()),
		)
	}
}

func withoutRetired(
	item *billingqueue.BillingQueueItem,
	retired map[pulid.ID]pulid.ID,
) ([]pulid.ID, []billingqueue.PacketPageRotation, bool) {
	if len(retired) == 0 || len(item.PacketDocumentIDs) == 0 {
		return item.PacketDocumentIDs, item.PacketPageRotations, false
	}
	changed := false
	packet := make([]pulid.ID, 0, len(item.PacketDocumentIDs))
	for _, id := range item.PacketDocumentIDs {
		replacement, gone := retired[id]
		if !gone {
			packet = append(packet, id)
			continue
		}
		changed = true
		if !replacement.IsNil() && !slices.Contains(item.PacketDocumentIDs, replacement) &&
			!slices.Contains(packet, replacement) {
			packet = append(packet, replacement)
		}
	}
	if !changed {
		return item.PacketDocumentIDs, item.PacketPageRotations, false
	}
	rotations := make([]billingqueue.PacketPageRotation, 0, len(item.PacketPageRotations))
	for _, turn := range item.PacketPageRotations {
		if _, gone := retired[turn.DocumentID]; !gone {
			rotations = append(rotations, turn)
		}
	}

	return packet, rotations, true
}
