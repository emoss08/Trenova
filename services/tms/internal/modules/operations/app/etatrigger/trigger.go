package etatrigger

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Requester stopetaports.RefreshRequester
}

type Trigger struct {
	requester stopetaports.RefreshRequester
}

var (
	_ shipmenteventports.ShipmentEventObserver = (*Trigger)(nil)
	_ checkcallports.CheckCallObserver         = (*Trigger)(nil)
)

func New(p Params) *Trigger {
	return &Trigger{requester: p.Requester}
}

func Refreshes(eventType shipmentevent.Type) bool {
	switch eventType {
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeStopCompleted,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned,
		shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned,
		shipmentevent.TypeTenderAccepted:
		return true
	case shipmentevent.TypeOwnershipTransferred,
		shipmentevent.TypeTenderOffered,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.TypeTenderExpired,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.TypeTenderEntryWarned,
		shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldUpdated,
		shipmentevent.TypeHoldReleased,
		shipmentevent.TypeCommentPosted:
		return false
	default:
		return false
	}
}

func (t *Trigger) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if event == nil || !Refreshes(event.Type) {
		return nil
	}
	t.requester.RequestRefresh(ctx, pagination.TenantInfo{
		OrgID: event.OrganizationID,
		BuID:  event.BusinessUnitID,
	}, event.ShipmentID)
	return nil
}

func (t *Trigger) CheckCallRecorded(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID, _ pulid.ID,
) {
	t.requester.RequestRefresh(ctx, tenantInfo, shipmentID)
}
