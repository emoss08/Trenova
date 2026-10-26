package customerstatustrigger

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Requester customertrackingports.EvaluationRequester
	Facts     customertrackingports.FactRepository
	Logger    *zap.Logger
}

// Trigger asks for a shipment's status emails to be evaluated when it changes
// in a way that could owe its customer one: a stop is reached or left, its
// status moves, or its projected arrivals are refreshed. Organizations with no
// customer rule on are skipped without starting anything.
type Trigger struct {
	requester customertrackingports.EvaluationRequester
	facts     customertrackingports.FactRepository
	l         *zap.Logger
}

var (
	_ shipmenteventports.ShipmentEventObserver = (*Trigger)(nil)
	_ stopetaports.Observer                    = (*Trigger)(nil)
)

func New(p Params) *Trigger {
	return &Trigger{
		requester: p.Requester,
		facts:     p.Facts,
		l:         p.Logger.Named("customer-status-trigger"),
	}
}

func Evaluates(eventType shipmentevent.Type) bool {
	switch eventType {
	case shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeStopCompleted:
		return true
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned,
		shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned,
		shipmentevent.TypeOwnershipTransferred,
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
	if event == nil || !Evaluates(event.Type) {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: event.OrganizationID, BuID: event.BusinessUnitID}
	if !t.notifies(ctx, tenant) {
		return nil
	}
	t.requester.RequestEvaluation(ctx, tenant, event.ShipmentID)
	return nil
}

func (t *Trigger) StopEtasRefreshed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) {
	if len(shipmentIDs) == 0 || !t.notifies(ctx, tenantInfo) {
		return
	}
	for _, shipmentID := range shipmentIDs {
		t.requester.RequestEvaluation(ctx, tenantInfo, shipmentID)
	}
}

func (t *Trigger) notifies(ctx context.Context, tenant pagination.TenantInfo) bool {
	notifying, err := t.facts.HasNotifyingCustomers(ctx, tenant)
	if err != nil {
		t.l.Debug("could not read whether the organization sends status emails",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err))
		return true
	}
	return notifying
}
