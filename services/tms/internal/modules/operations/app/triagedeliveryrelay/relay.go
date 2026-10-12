package triagedeliveryrelay

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

var (
	_ shipmenteventports.ShipmentEventObserver = (*Relay)(nil)
	_ shipmentports.MoveStatusObserver         = (*Relay)(nil)
)

type shipmentReader interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
}

type Params struct {
	fx.In

	Shipments shipmentports.ShipmentRepository
	Sink      inboundmessageports.TriageEventSink `optional:"true"`
}

type Relay struct {
	shipments shipmentReader
	sink      inboundmessageports.TriageEventSink
}

func New(p Params) *Relay {
	return &Relay{shipments: p.Shipments, sink: p.Sink}
}

func Delivered(status shipment.Status) bool {
	switch status {
	case shipment.StatusCompleted, shipment.StatusReadyToInvoice, shipment.StatusInvoiced:
		return true
	case shipment.StatusNew,
		shipment.StatusPartiallyAssigned,
		shipment.StatusAssigned,
		shipment.StatusInTransit,
		shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
		shipment.StatusCanceled:
		return false
	default:
		return false
	}
}

func (r *Relay) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if r.sink == nil || event == nil || event.Type != shipmentevent.TypeStatusChanged {
		return nil
	}

	next := statusIn(event.Metadata, "newStatus")
	previous := statusIn(event.Metadata, "previousStatus")
	if !Delivered(next) || Delivered(previous) {
		return nil
	}

	r.sink.ShipmentDelivered(ctx, pagination.TenantInfo{
		OrgID: event.OrganizationID,
		BuID:  event.BusinessUnitID,
	}, event.ShipmentID)

	return nil
}

func (r *Relay) AfterMoveStatusChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	previous shipment.MoveStatus,
) error {
	if r.sink == nil || move == nil || move.Status != shipment.MoveStatusCompleted ||
		previous == shipment.MoveStatusCompleted {
		return nil
	}

	entity, err := r.shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if !Delivered(entity.Status) {
		return nil
	}

	r.sink.ShipmentDelivered(ctx, tenantInfo, move.ShipmentID)

	return nil
}

func statusIn(metadata map[string]any, key string) shipment.Status {
	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}

	return shipment.Status(value)
}
