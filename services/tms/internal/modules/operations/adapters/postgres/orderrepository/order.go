package orderrepository

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrOrderChargeRequestNil   = errors.New("request cannot be nil")
	ErrOrderChargeRequestEmpty = errors.New("request cannot be empty")
)

var (
	orderErrors       = dberror.Mapping{Entity: "Order"}
	orderChargeErrors = dberror.Mapping{Entity: "Order charge"}
)

type Params struct {
	fx.In

	DB         *postgres.Connection
	Logger     *zap.Logger
	References orderports.OrderReferences
}

type repository struct {
	db         *postgres.Connection
	l          *zap.Logger
	references orderports.OrderReferences
}

func New(p Params) orderports.OrderRepository {
	return &repository{
		db:         p.DB,
		l:          p.Logger.Named("postgres.order-repository"),
		references: p.References,
	}
}

func (r *repository) List(
	ctx context.Context,
	req *orderports.ListOrdersRequest,
) (*pagination.ListResult[*order.Order], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*order.Order], error) {
			o := orderdb.Orders
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				o.Query(),
				&listquery.Request[order.Order]{
					Filter: req.Filter,
					Then:   []keel.Order{o.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list orders", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *orderports.ListOrdersConnectionRequest,
) (*pagination.CursorListResult[*order.Order], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*order.Order], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				orderdb.Orders.Query(),
				&listquery.Request[order.Order]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.OrderColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list orders", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *order.Order,
) (*order.Order, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*order.Order, error) {
		if err := orderdb.Orders.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create order",
				zap.String("orderNumber", entity.OrderNumber),
				zap.Error(err),
			)
			return nil, orderErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *order.Order,
) (*order.Order, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*order.Order, error) {
		o := orderdb.Orders
		if err := o.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			o.CustomerID,
			o.OwnerID,
			o.PONumber,
			o.BOL,
			o.CurrencyCode,
			o.QuotedAmount,
			o.BaseAmount,
		); err != nil {
			r.l.Error(
				"failed to update order",
				zap.String("orderNumber", entity.OrderNumber),
				zap.Error(err),
			)
			return nil, orderErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *orderports.UpdateOrderStatusRequest,
) (*order.Order, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*order.Order, error) {
		o := orderdb.Orders
		rows, err := o.Where(o.ID.Eq(req.OrderID), o.Version.Eq(req.Version)).
			UpdateReturning(ctx, r.db.Keel(ctx), o.Status.To(req.Status))
		if err != nil {
			r.l.Error(
				"failed to update order status",
				zap.String("id", req.OrderID.String()),
				zap.Error(err),
			)
			return nil, orderErrors.Map(err, req.OrderID.String())
		}

		if err = dberror.CheckAffected(
			int64(len(rows)),
			"Order",
			req.OrderID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) GetShipmentStatuses(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
) ([]shipment.Status, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]shipment.Status, error) {
		sp := shipmentdb.Shipments
		statuses, err := keel.Into[shipment.Status](
			ctx,
			r.db.Keel(ctx),
			sp.Pick(sp.Status).Where(sp.OrderID.Eq(orderID)),
		)
		if err != nil {
			r.l.Error("failed to get shipment statuses for order", zap.Error(err))
			return nil, err
		}
		if statuses == nil {
			statuses = []shipment.Status{}
		}

		return statuses, nil
	})
}

func unchangedShipmentVersion() []keel.Assignment {
	sp := shipmentdb.Shipments
	return []keel.Assignment{sp.Version.ToExpr(sp.Version), sp.UpdatedAt.ToExpr(sp.UpdatedAt)}
}

func (r *repository) AttachShipments(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
	shipmentIDs []pulid.ID,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if len(shipmentIDs) == 0 {
			return 0, nil
		}

		sp := shipmentdb.Shipments
		affected, err := sp.Where(
			sp.ID.In(shipmentIDs...),
			sp.Status.NotIn(shipment.StatusCanceled, shipment.StatusInvoiced),
		).Update(
			ctx,
			r.db.Keel(ctx),
			append([]keel.Assignment{sp.OrderID.To(orderID)}, unchangedShipmentVersion()...)...,
		)
		if err != nil {
			r.l.Error("failed to attach shipments to order", zap.Error(err))
			return 0, err
		}

		return affected, nil
	})
}

func (r *repository) DetachShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
	shipmentID pulid.ID,
	newOrderID pulid.ID,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		sp := shipmentdb.Shipments
		affected, err := sp.Where(
			sp.ID.Eq(shipmentID),
			sp.OrderID.Eq(orderID),
			sp.Status.Ne(shipment.StatusInvoiced),
		).Update(
			ctx,
			r.db.Keel(ctx),
			append([]keel.Assignment{sp.OrderID.To(newOrderID)}, unchangedShipmentVersion()...)...,
		)
		if err != nil {
			r.l.Error("failed to detach shipment from order", zap.Error(err))
			return 0, err
		}

		return affected, nil
	})
}

func (r *repository) GetShipmentAttachRefs(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]orderports.ShipmentAttachRef, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]orderports.ShipmentAttachRef, error) {
		if len(shipmentIDs) == 0 {
			return nil, nil
		}

		sp := shipmentdb.Shipments
		refs, err := keel.Into[orderports.ShipmentAttachRef](ctx, r.db.Keel(ctx), sp.
			Pick(sp.ID, keel.DynOf(sp.OrderID).Coalesce(""), sp.CustomerID, sp.Status).
			Where(sp.ID.In(shipmentIDs...)).
			Limit(len(shipmentIDs)))
		if err != nil {
			r.l.Error("failed to load shipment attach refs", zap.Error(err))
			return nil, err
		}

		return refs, nil
	})
}

func (r *repository) DeleteIfEmpty(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		deleted, err := r.emptyOrder(orderID).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete empty order", zap.Error(err))
			return 0, err
		}

		return deleted, nil
	})
}

func (r *repository) emptyOrder(orderID pulid.ID) *keel.Query[*order.Order] {
	o := orderdb.Orders
	preds := make([]keel.Pred, 0, 3+len(r.references))
	preds = append(preds, o.ID.Eq(orderID), o.Shipments().None(), o.Charges().None())
	for _, ref := range r.references {
		preds = append(
			preds,
			keel.NotExists(keel.Handle[*keel.Row]{Ref: ref.Ref}.Where(ref.EqCol(o.ID))),
		)
	}
	return o.Where(preds...)
}

func (r *repository) AddCharge(
	ctx context.Context,
	entity *order.OrderCharge,
) (*order.OrderCharge, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*order.OrderCharge, error) {
		if err := orderdb.OrderCharges.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to add order charge", zap.Error(err))
			return nil, orderChargeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func noInvoicedShare() keel.Pred {
	oc := orderdb.OrderCharges
	a := oc.Allocations()

	return a.None(a.InvoiceID.IsNotNull())
}

func (r *repository) RemoveCharge(
	ctx context.Context,
	req *orderports.RemoveOrderChargeRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil {
			return 0, ErrOrderChargeRequestNil
		}

		oc := orderdb.OrderCharges
		deleted, err := oc.Where(
			oc.ID.Eq(req.ChargeID),
			oc.OrderID.Eq(req.OrderID),
			oc.InvoiceID.IsNull(),
			noInvoicedShare(),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to remove order charge", zap.Error(err))
			return 0, err
		}

		return deleted, nil
	})
}

func (r *repository) UpdateCharge(
	ctx context.Context,
	entity *order.OrderCharge,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		oc := orderdb.OrderCharges
		rows, err := oc.Where(
			oc.ID.Eq(entity.ID),
			oc.Version.Eq(entity.Version),
			oc.OrderID.Eq(entity.OrderID),
			oc.InvoiceID.IsNull(),
			noInvoicedShare(),
		).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			oc.Description.To(entity.Description),
			oc.Amount.To(entity.Amount),
		)
		if err != nil {
			r.l.Error("failed to update order charge", zap.Error(err))
			return 0, orderChargeErrors.Map(err, entity.ID.String())
		}

		if len(rows) == 1 {
			allocations := entity.Allocations
			*entity = *rows[0]
			entity.Allocations = allocations
		}

		return int64(len(rows)), nil
	})
}

func (r *repository) ListCharges(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*order.OrderCharge, error) {
		oc := orderdb.OrderCharges
		charges, err := oc.Where(oc.OrderID.Eq(orderID)).
			Order(oc.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list order charges", zap.Error(err))
			return nil, err
		}

		return nonNil(charges), nil
	})
}

func (r *repository) ListUninvoicedCharges(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*order.OrderCharge, error) {
		oc := orderdb.OrderCharges
		charges, err := oc.Where(oc.OrderID.Eq(orderID), oc.InvoiceID.IsNull()).
			Order(oc.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list uninvoiced order charges", zap.Error(err))
			return nil, err
		}

		return nonNil(charges), nil
	})
}

func (r *repository) ListUninvoicedChargeSharesForPayer(
	ctx context.Context,
	req *orderports.ListUninvoicedChargeSharesRequest,
) ([]*order.OrderCharge, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*order.OrderCharge, error) {
		if req == nil {
			return nil, ErrOrderChargeRequestNil
		}

		oc := orderdb.OrderCharges
		a := oc.Allocations()
		charges, err := oc.Where(
			oc.OrderID.Eq(req.OrderID),
			oc.InvoicedAt.IsNull(),
			keel.Or(
				keel.And(oc.InvoiceID.IsNull(), a.None()),
				a.Any(a.BillToCustomerID.Eq(req.PayerID), a.InvoiceID.IsNull()),
			),
		).
			Load(a.Order(a.Sequence.Asc(), a.ID.Asc())).
			Order(oc.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list uninvoiced order charge shares", zap.Error(err))
			return nil, err
		}

		return nonNil(charges), nil
	})
}

func unchangedChargeVersion() keel.Assignment {
	oc := orderdb.OrderCharges
	return oc.Version.ToExpr(oc.Version)
}

func (r *repository) ClearChargesInvoice(
	ctx context.Context,
	req *orderports.ClearOrderChargesInvoiceRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil || req.InvoiceID.IsNil() {
			return 0, nil
		}

		oc := orderdb.OrderCharges
		cleared, err := oc.Where(oc.InvoiceID.Eq(req.InvoiceID)).Update(
			ctx,
			r.db.Keel(ctx),
			oc.InvoiceID.ToNull(),
			oc.InvoicedAt.ToNull(),
			unchangedChargeVersion(),
		)
		if err != nil {
			r.l.Error("failed to clear order charges invoice", zap.Error(err))
			return 0, err
		}

		return cleared, nil
	})
}

func (r *repository) MarkChargesFullyInvoicedWhereComplete(
	ctx context.Context,
	req *orderports.MarkChargesFullyInvoicedRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil {
			return 0, ErrOrderChargeRequestEmpty
		}

		oc := orderdb.OrderCharges
		a := oc.Allocations()
		marked, err := oc.Where(
			oc.OrderID.Eq(req.OrderID),
			oc.InvoicedAt.IsNull(),
			a.Any(),
			a.None(a.InvoiceID.IsNull()),
		).Update(
			ctx,
			r.db.Keel(ctx),
			oc.InvoicedAt.To(req.InvoicedAt),
			oc.InvoiceID.ToExpr(keel.DynOf(oc.InvoiceID).Coalesce(req.InvoiceID.String())),
			unchangedChargeVersion(),
		)
		if err != nil {
			r.l.Error("failed to mark order charges fully invoiced", zap.Error(err))
			return 0, err
		}

		return marked, nil
	})
}

func (r *repository) MarkChargesInvoiced(
	ctx context.Context,
	req *orderports.MarkOrderChargesInvoicedRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req == nil {
			return 0, ErrOrderChargeRequestEmpty
		}

		if len(req.ChargeIDs) == 0 {
			return 0, nil
		}

		oc := orderdb.OrderCharges
		marked, err := oc.Where(
			oc.OrderID.Eq(req.OrderID),
			oc.ID.In(req.ChargeIDs...),
			oc.InvoiceID.IsNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			oc.InvoiceID.To(req.InvoiceID),
			oc.InvoicedAt.To(req.InvoicedAt),
			unchangedChargeVersion(),
		)
		if err != nil {
			r.l.Error("failed to mark order charges invoiced", zap.Error(err))
			return 0, err
		}

		return marked, nil
	})
}

func (r *repository) RecalculateTotal(
	ctx context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		o := orderdb.Orders
		sp := shipmentdb.Shipments.As("leg")
		oc := orderdb.OrderCharges.As("leg_charge")

		legTotal := keel.Scalar(sp.Pick(keel.DynOf(sp.TotalChargeAmount).Sum().Coalesce(0)).
			Where(sp.OrderID.Eq(orderID), sp.Status.Ne(shipment.StatusCanceled)))
		chargeTotal := keel.Scalar(oc.Pick(keel.DynOf(oc.Amount).Sum().Coalesce(0)).
			Where(oc.OrderID.Eq(orderID)))

		if _, err := o.Where(o.ID.Eq(orderID)).Update(
			ctx,
			r.db.Keel(ctx),
			o.TotalAmount.ToExpr(keel.DynOf(legTotal).Add(chargeTotal)),
		); err != nil {
			r.l.Error("failed to recalculate order total", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req orderports.GetOrderByIDRequest,
) (*order.Order, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*order.Order, error) {
		o := orderdb.Orders
		loads := make([]keel.Loader, 0, 3)
		if req.IncludeShipment {
			loads = append(loads, o.Shipments())
		}
		if req.IncludeShipment || req.IncludeCustomer {
			loads = append(loads, o.Customer())
		}
		if req.IncludeShipment || req.IncludeCharges {
			loads = append(loads, o.Charges())
		}

		entity, err := o.Where(o.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get order", zap.String("id", req.ID.String()), zap.Error(err))
			return nil, orderErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req orderports.GetOrdersByIDsRequest,
) ([]*order.Order, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*order.Order, error) {
		if len(req.OrderIDs) == 0 {
			return []*order.Order{}, nil
		}

		o := orderdb.Orders
		entities, err := o.Where(o.ID.In(req.OrderIDs...)).
			Limit(len(req.OrderIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get orders", zap.Error(err))
			return nil, orderErrors.Map(err, "")
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *orderports.OrderSelectOptionsRequest,
) (*pagination.ListResult[*order.Order], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*order.Order], error) {
			o := orderdb.Orders
			q := o.Pick(o.ID, o.OrderNumber, o.Status)
			if req.AttachableOnly {
				q = q.Where(
					o.Status.NotIn(order.StatusBilled, order.StatusClosed, order.StatusCanceled),
				)
			}
			if !req.CustomerID.IsNil() {
				q = q.Where(o.CustomerID.Eq(req.CustomerID))
			}

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						o.OrderNumber.ContainsFold(s),
						o.PONumber.ContainsFold(s),
						o.BOL.ContainsFold(s),
					)
				},
			)
		},
	)
}

func nonNil[T any](items []*T) []*T {
	if items == nil {
		return []*T{}
	}

	return items
}
