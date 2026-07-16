package orderrepository

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrOrderChargeRequestNil   = errors.New("request cannot be nil")
	ErrOrderChargeRequestEmpty = errors.New("request cannot be empty")
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.OrderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.order-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListOrdersRequest,
) *bun.SelectQuery {
	cols := buncolgen.OrderColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.OrderTable.Alias,
		req.Filter,
		(*order.Order)(nil),
	)

	return q.Apply(buncolgen.OrderApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListOrdersRequest,
) (*pagination.ListResult[*order.Order], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*order.Order, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count orders", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*order.Order]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListOrdersConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.OrderTable.Alias,
		req.Filter,
		req.Cursor,
		(*order.Order)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListOrdersConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.OrderTable.Alias,
		req.Filter,
		(*order.Order)(nil),
	)
}

func applyOrderColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.OrderTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListOrdersConnectionRequest,
) (*pagination.CursorListResult[*order.Order], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*order.Order)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count orders", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*order.Order]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*order.Order) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyOrderColumns(sq, req.OrderColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan orders", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *order.Order,
) (*order.Order, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orderNumber", entity.OrderNumber),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create order", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) CreateInTx(
	ctx context.Context,
	tx bun.IDB,
	entity *order.Order,
) error {
	if _, err := tx.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		r.l.Error("failed to create order in tx", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *order.Order,
) (*order.Order, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orderNumber", entity.OrderNumber),
	)

	ov := entity.Version
	entity.Version++
	cols := buncolgen.OrderColumns

	// Explicit column list: cleared optional fields (PO, BOL, owner, quote amounts)
	// must persist as NULL rather than being dropped from the SET clause, and the
	// derived columns (status, total_amount) plus the immutable order_number are
	// never written through the generic update.
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column(
			cols.CustomerID.Name,
			cols.OwnerID.Name,
			cols.PONumber.Name,
			cols.BOL.Name,
			cols.CurrencyCode.Name,
			cols.QuotedAmount.Name,
			cols.BaseAmount.Name,
			cols.Version.Name,
			cols.UpdatedAt.Name,
		).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update order", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Order", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateOrderStatusRequest,
) (*order.Order, error) {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("id", req.OrderID.String()),
	)

	entity := &order.Order{
		ID:             req.OrderID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         req.Status,
		Version:        req.Version + 1,
	}
	cols := buncolgen.OrderColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), req.Version).
		Set(cols.Status.Set(), req.Status).
		Set(cols.Version.Set(), req.Version+1).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update order status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Order", req.OrderID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetShipmentStatuses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) ([]shipment.Status, error) {
	statuses := make([]shipment.Status, 0)

	cols := buncolgen.ShipmentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column("status").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, tenantInfo).
				Where(cols.OrderID.Eq(), orderID)
		}).
		Scan(ctx, &statuses)
	if err != nil {
		r.l.Error("failed to get shipment statuses for order", zap.Error(err))
		return nil, err
	}

	return statuses, nil
}

func (r *repository) AttachShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentIDs []pulid.ID,
) (int64, error) {
	if len(shipmentIDs) == 0 {
		return 0, nil
	}

	cols := buncolgen.ShipmentColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("order_id = ?", orderID).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentScopeTenantUpdate(sq, tenantInfo).
				Where(cols.ID.In(), bun.List(shipmentIDs)).
				Where(cols.Status.Ne(), shipment.StatusCanceled).
				Where(cols.Status.Ne(), shipment.StatusInvoiced)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to attach shipments to order", zap.Error(err))
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return affected, nil
}

func (r *repository) DetachShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentID pulid.ID,
	newOrderID pulid.ID,
) (int64, error) {
	cols := buncolgen.ShipmentColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("order_id = ?", newOrderID).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.Eq(), shipmentID).
				Where(cols.OrderID.Eq(), orderID).
				Where(cols.Status.Ne(), shipment.StatusInvoiced)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to detach shipment from order", zap.Error(err))
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return affected, nil
}

func (r *repository) GetShipmentAttachRefs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]repositories.ShipmentAttachRef, error) {
	if len(shipmentIDs) == 0 {
		return nil, nil
	}

	refs := make([]repositories.ShipmentAttachRef, 0, len(shipmentIDs))
	cols := buncolgen.ShipmentColumns
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column("id", "order_id", "customer_id", "status").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, tenantInfo).
				Where(cols.ID.In(), bun.List(shipmentIDs))
		}).
		Scan(ctx, &refs)
	if err != nil {
		r.l.Error("failed to load shipment attach refs", zap.Error(err))
		return nil, err
	}

	return refs, nil
}

func (r *repository) DeleteIfEmpty(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) (int64, error) {
	cols := buncolgen.OrderColumns
	result, err := r.db.DBForContext(ctx).NewDelete().
		Model((*order.Order)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.OrderScopeTenantDelete(dq, tenantInfo).
				Where(cols.ID.Eq(), orderID)
		}).
		Where("NOT EXISTS (SELECT 1 FROM shipments s WHERE s.order_id = ord.id AND s.organization_id = ord.organization_id AND s.business_unit_id = ord.business_unit_id)").
		Where("NOT EXISTS (SELECT 1 FROM order_charges oc WHERE oc.order_id = ord.id AND oc.organization_id = ord.organization_id AND oc.business_unit_id = ord.business_unit_id)").
		Where("NOT EXISTS (SELECT 1 FROM invoices inv WHERE inv.order_id = ord.id AND inv.organization_id = ord.organization_id AND inv.business_unit_id = ord.business_unit_id)").
		Where("NOT EXISTS (SELECT 1 FROM billing_queue_items bqi WHERE bqi.order_id = ord.id AND bqi.organization_id = ord.organization_id AND bqi.business_unit_id = ord.business_unit_id)").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete empty order", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}

func (r *repository) AddCharge(
	ctx context.Context,
	entity *order.OrderCharge,
) (*order.OrderCharge, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to add order charge", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) RemoveCharge(
	ctx context.Context,
	req *repositories.RemoveOrderChargeRequest,
) (int64, error) {
	if req == nil {
		return 0, ErrOrderChargeRequestNil
	}

	cols := buncolgen.OrderChargeColumns
	result, err := r.db.DBForContext(ctx).NewDelete().
		Model((*order.OrderCharge)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.OrderChargeScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ChargeID).
				Where(cols.OrderID.Eq(), req.OrderID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to remove order charge", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}

func (r *repository) UpdateCharge(
	ctx context.Context,
	entity *order.OrderCharge,
) (int64, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.OrderChargeColumns

	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		Column(
			cols.Description.Name,
			cols.Amount.Name,
			cols.Version.Name,
			cols.UpdatedAt.Name,
		).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Where(cols.OrderID.Eq(), entity.OrderID).
		Where(cols.InvoiceID.IsNull()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update order charge", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}

func (r *repository) ListCharges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	charges := make([]*order.OrderCharge, 0)

	cols := buncolgen.OrderChargeColumns
	err := r.db.DBForContext(ctx).NewSelect().
		Model(&charges).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OrderChargeScopeTenant(sq, tenantInfo).
				Where(cols.OrderID.Eq(), orderID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list order charges", zap.Error(err))
		return nil, err
	}

	return charges, nil
}

func (r *repository) ListUninvoicedCharges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	charges := make([]*order.OrderCharge, 0)

	cols := buncolgen.OrderChargeColumns
	err := r.db.DBForContext(ctx).NewSelect().
		Model(&charges).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OrderChargeScopeTenant(sq, tenantInfo).
				Where(cols.OrderID.Eq(), orderID).
				Where(cols.InvoiceID.IsNull())
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list uninvoiced order charges", zap.Error(err))
		return nil, err
	}

	return charges, nil
}

func (r *repository) MarkChargesInvoiced(
	ctx context.Context,
	req *repositories.MarkOrderChargesInvoicedRequest,
) (int64, error) {
	if req == nil {
		return 0, ErrOrderChargeRequestEmpty
	}

	if len(req.ChargeIDs) == 0 {
		return 0, nil
	}

	cols := buncolgen.OrderChargeColumns
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*order.OrderCharge)(nil)).
		Set(cols.InvoiceID.Set(), req.InvoiceID).
		Set(cols.InvoicedAt.Set(), req.InvoicedAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.OrderChargeScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.OrderID.Eq(), req.OrderID).
				Where(cols.ID.In(), bun.List(req.ChargeIDs)).
				Where(cols.InvoiceID.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark order charges invoiced", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}

func (r *repository) RecalculateTotal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) error {
	cols := buncolgen.OrderColumns
	sc := buncolgen.ShipmentColumns
	cc := buncolgen.OrderChargeColumns
	db := r.db.DBForContext(ctx)

	legTotal := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("COALESCE(SUM(?), 0)", bun.Ident(sc.TotalChargeAmount.Name)).
		Apply(buncolgen.ShipmentApplyTenant(tenantInfo)).
		Where(sc.OrderID.Eq(), orderID).
		Where(sc.Status.Ne(), shipment.StatusCanceled)

	chargeTotal := db.NewSelect().
		Model((*order.OrderCharge)(nil)).
		ColumnExpr("COALESCE(SUM(?), 0)", bun.Ident(cc.Amount.Name)).
		Apply(buncolgen.OrderChargeApplyTenant(tenantInfo)).
		Where(cc.OrderID.Eq(), orderID)

	_, err := db.NewUpdate().
		Model((*order.Order)(nil)).
		Set(cols.TotalAmount.SetExpr("(?) + (?)"), legTotal, chargeTotal).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.OrderScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.Eq(), orderID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to recalculate order total", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetOrderByIDRequest,
) (*order.Order, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(order.Order)
	cols := buncolgen.OrderColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OrderScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})

	if req.IncludeShipment {
		q = q.Relation("Shipments").Relation("Customer").Relation("Charges")
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get order", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Order")
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetOrdersByIDsRequest,
) ([]*order.Order, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*order.Order, 0, len(req.OrderIDs))
	cols := buncolgen.OrderColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OrderScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.OrderIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get orders", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Order")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.OrderSelectOptionsRequest,
) (*pagination.ListResult[*order.Order], error) {
	cols := buncolgen.OrderColumns

	return dbhelper.SelectOptions[*order.Order](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.OrderNumber,
				cols.Status,
			},
			OrgColumnRef:     &cols.OrganizationID,
			BuColumnRef:      &cols.BusinessUnitID,
			EntityName:       "Order",
			SearchColumnRefs: []buncolgen.Column{cols.OrderNumber, cols.PONumber, cols.BOL},
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if req.AttachableOnly {
					q = q.Where(cols.Status.Ne(), order.StatusBilled).
						Where(cols.Status.Ne(), order.StatusClosed).
						Where(cols.Status.Ne(), order.StatusCanceled)
				}
				if !req.CustomerID.IsNil() {
					q = q.Where(cols.CustomerID.Eq(), req.CustomerID)
				}
				return q
			},
		},
	)
}
