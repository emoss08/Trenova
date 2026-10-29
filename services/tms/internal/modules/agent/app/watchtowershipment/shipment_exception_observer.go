package watchtowershipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ShipmentExceptionObserverParams struct {
	fx.In

	ShipmentExceptionParams

	Reconciler watchtowerports.WatchtowerSubjectReconciler
	Logger     *zap.Logger
}

// ShipmentExceptionObserver settles a shipment's missed stops, missing POD
// and exhausted tender the moment something happens to the shipment, so an
// arrival or a covering carrier takes the item off the queue without waiting
// for the sweep. The sweep still raises what only the clock can.
type ShipmentExceptionObserver struct {
	exceptions shipmentExceptions
	reconciler watchtowerports.WatchtowerSubjectReconciler
	l          *zap.Logger
}

var _ shipmenteventports.ShipmentEventObserver = (*ShipmentExceptionObserver)(nil)

func NewShipmentExceptionObserver(
	p ShipmentExceptionObserverParams,
) shipmenteventports.ShipmentEventObserver {
	return &ShipmentExceptionObserver{
		exceptions: newShipmentExceptions(p.ShipmentExceptionParams),
		reconciler: p.Reconciler,
		l:          p.Logger.Named("watchtower.shipment-exceptions"),
	}
}

func ShipmentExceptionKindsFor(eventType shipmentevent.Type) []watchtower.SourceKind {
	switch eventType {
	case shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeOwnershipTransferred:
		return ShipmentExceptionKinds()
	case shipmentevent.TypeMoveArrived,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeStopCompleted:
		return []watchtower.SourceKind{
			watchtower.SourceMissedPickup,
			watchtower.SourceMissedDelivery,
			watchtower.SourcePodMissing,
		}
	case shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.TypeTenderBounced,
		shipmentevent.TypeTenderEscalated,
		shipmentevent.TypeTenderOffered,
		shipmentevent.TypeTenderAccepted,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned:
		return []watchtower.SourceKind{watchtower.SourceTenderExhausted}
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.TypeTenderExpired,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.TypeTenderEntryWarned,
		shipmentevent.TypeTenderCountered,
		shipmentevent.TypeTenderCounterDecided,
		shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldUpdated,
		shipmentevent.TypeHoldReleased,
		shipmentevent.TypeCommentPosted:
		return nil
	}
	return nil
}

func (o *ShipmentExceptionObserver) OnShipmentEvent(
	ctx context.Context,
	event *shipmentevent.Event,
) error {
	if event == nil || event.ShipmentID.IsNil() {
		return nil
	}
	kinds := ShipmentExceptionKindsFor(event.Type)
	if len(kinds) == 0 {
		return nil
	}

	tenant := pagination.TenantInfo{OrgID: event.OrganizationID, BuID: event.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	shipmentIDs := []pulid.ID{event.ShipmentID}
	for _, kind := range kinds {
		open, err := o.exceptions.forKind(ctx, tenant, kind, shipmentIDs)
		if err != nil {
			o.l.Warn("could not settle a shipment's exceptions",
				zap.String("kind", string(kind)),
				zap.String("shipment", event.ShipmentID.String()),
				zap.Error(err))
			continue
		}
		o.reconciler.ReconcileSubject(ctx, &watchtowerports.ReconcileSubjectRequest{
			TenantInfo: tenant,
			SourceKind: kind,
			SubjectID:  event.ShipmentID,
			Open:       open,
		})
	}
	return nil
}
