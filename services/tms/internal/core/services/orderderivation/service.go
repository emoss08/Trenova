package orderderivation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxRecomputeAttempts = 4

var (
	_ services.ShipmentEventObserver  = (*Observer)(nil)
	_ services.OrderDerivationService = (*Observer)(nil)
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	ShipmentRepo repositories.ShipmentRepository
	OrderRepo    repositories.OrderRepository
	Realtime     services.RealtimeService
}

type Observer struct {
	l            *zap.Logger
	shipmentRepo shipmentReader
	orderRepo    repositories.OrderRepository
	realtime     services.RealtimeService
}

type shipmentReader interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
}

func New(p Params) *Observer {
	return &Observer{
		l:            p.Logger.Named("service.order-derivation"),
		shipmentRepo: p.ShipmentRepo,
		orderRepo:    p.OrderRepo,
		realtime:     p.Realtime,
	}
}

// OnShipmentEvent recomputes the parent order's status whenever one of its legs
// changes status. The order status is derived from the full set of leg statuses via
// the pure order.Derive function, so the recompute is idempotent and converges
// regardless of the order in which concurrent leg events arrive.
func (o *Observer) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if event == nil || !derivationRelevant(event.Type) {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: event.OrganizationID,
		BuID:  event.BusinessUnitID,
	}

	return o.RecomputeForShipment(ctx, tenantInfo, event.ShipmentID)
}

// RecomputeForShipment resolves the shipment's parent order and recomputes its
// derived state. A shipment with no order is a no-op.
func (o *Observer) RecomputeForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	shp, err := o.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		o.l.Error("failed to load shipment for order derivation", zap.Error(err))
		return err
	}
	if shp.OrderID.IsNil() {
		return nil
	}

	return o.RecomputeOrder(ctx, tenantInfo, shp.OrderID)
}

// RecomputeOrder recalculates the order's AR total and re-derives its status from the
// current leg statuses, with bounded optimistic-conflict retries.
func (o *Observer) RecomputeOrder(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) error {
	if err := o.orderRepo.RecalculateTotal(ctx, tenantInfo, orderID); err != nil {
		o.l.Error(
			"failed to recalculate order total",
			zap.String("orderId", orderID.String()),
			zap.Error(err),
		)
		return err
	}

	var lastErr error
	for attempt := 0; attempt < maxRecomputeAttempts; attempt++ {
		ord, err := o.orderRepo.GetByID(ctx, repositories.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return err
		}

		// Closed is terminal and is set only by the manual close / AR settlement flow.
		if ord.Status == order.StatusClosed {
			return nil
		}

		statuses, err := o.orderRepo.GetShipmentStatuses(ctx, tenantInfo, orderID)
		if err != nil {
			return err
		}

		next := order.Derive(statuses)
		if next == ord.Status {
			return nil
		}

		updated, err := o.orderRepo.UpdateStatus(ctx, &repositories.UpdateOrderStatusRequest{
			TenantInfo: tenantInfo,
			OrderID:    orderID,
			Status:     next,
			Version:    ord.Version,
		})
		if err != nil {
			// Most likely an optimistic-version conflict from a concurrent leg event;
			// re-read and recompute.
			lastErr = err
			continue
		}

		o.publish(ctx, updated)
		return nil
	}

	o.l.Error(
		"exhausted attempts recomputing order status",
		zap.String("orderId", orderID.String()),
		zap.Error(lastErr),
	)
	return lastErr
}

func (o *Observer) publish(ctx context.Context, ord *order.Order) {
	if ord == nil {
		return
	}
	if err := realtimeinvalidation.Publish(
		ctx,
		o.realtime,
		&realtimeinvalidation.PublishParams{
			OrganizationID: ord.OrganizationID,
			BusinessUnitID: ord.BusinessUnitID,
			Resource:       "orders",
			Action:         "updated",
			RecordID:       ord.ID,
			Entity:         ord,
		},
	); err != nil {
		o.l.Warn("failed to publish order invalidation", zap.Error(err))
	}
}

func derivationRelevant(t shipmentevent.Type) bool {
	//nolint:exhaustive // only leg status-affecting events drive order derivation
	switch t {
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled:
		return true
	default:
		return false
	}
}
