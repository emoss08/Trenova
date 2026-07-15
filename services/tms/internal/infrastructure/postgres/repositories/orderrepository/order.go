package orderrepository

import (
	"context"

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
	total, err := r.db.DB().
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

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		OmitZero().
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

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), req.Version).
		Set(cols.Status.Set(), req.Status).
		Set(cols.Version.Set(), req.Version+1).
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
	err := r.db.DB().
		NewSelect().
		Table("shipments").
		Column("status").
		Where("order_id = ?", orderID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
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

	result, err := r.db.DBForContext(ctx).NewUpdate().
		Table("shipments").
		Set("order_id = ?", orderID).
		Where("id IN (?)", bun.List(shipmentIDs)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
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
) (int64, error) {
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Table("shipments").
		Set("order_id = NULL").
		Where("id = ?", shipmentID).
		Where("order_id = ?", orderID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
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

func (r *repository) CountShipmentsWithDifferentCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	shipmentIDs []pulid.ID,
) (int64, error) {
	if len(shipmentIDs) == 0 {
		return 0, nil
	}

	count, err := r.db.DB().NewSelect().
		Table("shipments").
		Where("id IN (?)", bun.List(shipmentIDs)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("customer_id != ?", customerID).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count shipments with different customer", zap.Error(err))
		return 0, err
	}

	return int64(count), nil
}

func (r *repository) AddCharge(
	ctx context.Context,
	entity *order.OrderCharge,
) (*order.OrderCharge, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		r.l.Error("failed to add order charge", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) RemoveCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	chargeID pulid.ID,
) (int64, error) {
	result, err := r.db.DB().NewDelete().
		Model((*order.OrderCharge)(nil)).
		Where("id = ?", chargeID).
		Where("order_id = ?", orderID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to remove order charge", zap.Error(err))
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
	err := r.db.DB().NewSelect().
		Model(&charges).
		Where("order_id = ?", orderID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list order charges", zap.Error(err))
		return nil, err
	}

	return charges, nil
}

func (r *repository) RecalculateTotal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) error {
	_, err := r.db.DB().NewUpdate().
		Model((*order.Order)(nil)).
		Set(`total_amount = COALESCE((SELECT SUM(s.total_charge_amount) FROM shipments s `+
			`WHERE s.order_id = ?0 AND s.organization_id = ?1 AND s.business_unit_id = ?2), 0) `+
			`+ COALESCE((SELECT SUM(c.amount) FROM order_charges c `+
			`WHERE c.order_id = ?0 AND c.organization_id = ?1 AND c.business_unit_id = ?2), 0)`,
			orderID, tenantInfo.OrgID, tenantInfo.BuID).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("id = ?", orderID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
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
	q := r.db.DB().
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
	err := r.db.DB().
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
		r.db.DB(),
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
		},
	)
}
