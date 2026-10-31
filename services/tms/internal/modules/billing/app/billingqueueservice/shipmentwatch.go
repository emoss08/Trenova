package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ShipmentObserverParams struct {
	fx.In

	Repo     billingqueueports.BillingQueueRepository
	Rechecks billingqueueports.RecheckRequester `optional:"true"`
	Logger   *zap.Logger
}

type ShipmentObserver struct {
	repo     billingqueueports.BillingQueueRepository
	rechecks billingqueueports.RecheckRequester
	l        *zap.Logger
}

var _ shipmenteventports.ShipmentEventObserver = (*ShipmentObserver)(nil)

func NewShipmentObserver(p ShipmentObserverParams) shipmenteventports.ShipmentEventObserver {
	return &ShipmentObserver{
		repo:     p.Repo,
		rechecks: p.Rechecks,
		l:        p.Logger.Named("billing-queue.shipment-observer"),
	}
}

func RechecksOn(eventType shipmentevent.Type) bool {
	switch eventType {
	case shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeStopCompleted,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldReleased:
		return true
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeOwnershipTransferred,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned,
		shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned,
		shipmentevent.TypeTenderOffered,
		shipmentevent.TypeTenderAccepted,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.TypeTenderExpired,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.TypeTenderEntryWarned,
		shipmentevent.TypeTenderBounced,
		shipmentevent.TypeTenderCountered,
		shipmentevent.TypeTenderCounterDecided,
		shipmentevent.TypeTenderEscalated,
		shipmentevent.TypeHoldUpdated,
		shipmentevent.TypeCommentPosted:
		return false
	default:
		return false
	}
}

func (o *ShipmentObserver) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if o.rechecks == nil || event == nil || event.ShipmentID.IsNil() || !RechecksOn(event.Type) {
		return nil
	}

	tenantInfo := pagination.TenantInfo{OrgID: event.OrganizationID, BuID: event.BusinessUnitID}
	byShipment, err := o.repo.ListActiveInvoiceItemsByShipmentIDs(
		ctx,
		&billingqueueports.ListActiveInvoiceItemsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: []pulid.ID{event.ShipmentID},
		},
	)
	if err != nil {
		o.l.Warn("could not find the billing items behind a shipment event",
			zap.String("shipmentId", event.ShipmentID.String()),
			zap.Error(err))
		return nil
	}

	for _, items := range byShipment {
		for _, item := range items {
			if item != nil {
				o.rechecks.RequestRecheck(ctx, tenantInfo, item.ID)
			}
		}
	}

	return nil
}
