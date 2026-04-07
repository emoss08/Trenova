package billingqueuerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.BillingQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListBillingQueueItemsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.BillingQueueItemTable.Alias,
		req.Filter,
		(*billingqueue.BillingQueueItem)(nil),
	)

	q = q.Relation(buncolgen.BillingQueueItemRelations.Shipment).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.Customer)).
		Relation(buncolgen.BillingQueueItemRelations.AssignedBiller).
		Relation(buncolgen.BillingQueueItemRelations.CanceledBy)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*billingqueue.BillingQueueItem, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count billing queue items", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*billingqueue.BillingQueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)
	entity := new(billingqueue.BillingQueueItem)

	if err := db.NewSelect().
		Model(entity).
		Where(bqi.ID.Eq(), req.ItemID).
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		Relation(buncolgen.BillingQueueItemRelations.Shipment).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.Customer)).
		Relation(buncolgen.Rel(buncolgen.BillingQueueItemRelations.Shipment, buncolgen.ShipmentRelations.AdditionalCharges)).
		Relation(buncolgen.BillingQueueItemRelations.AssignedBiller).
		Relation(buncolgen.BillingQueueItemRelations.CanceledBy).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Billing queue item")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert billing queue item: %w", err)
	}

	return r.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) (*billingqueue.BillingQueueItem, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	result, err := db.NewUpdate().
		Model(entity).
		Where(bqi.ID.Eq(), entity.ID).
		Where(bqi.OrganizationID.Eq(), entity.OrganizationID).
		Where(bqi.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(bqi.Version.Eq(), entity.Version).
		Set(bqi.Status.Set(), entity.Status).
		Set(bqi.AssignedBillerID.Set(), entity.AssignedBillerID).
		Set(bqi.ExceptionReasonCode.Set(), entity.ExceptionReasonCode).
		Set(bqi.ReviewNotes.Set(), entity.ReviewNotes).
		Set(bqi.ExceptionNotes.Set(), entity.ExceptionNotes).
		Set(bqi.ReviewStartedAt.Set(), entity.ReviewStartedAt).
		Set(bqi.ReviewCompletedAt.Set(), entity.ReviewCompletedAt).
		Set(bqi.CanceledByID.Set(), entity.CanceledByID).
		Set(bqi.CanceledAt.Set(), entity.CanceledAt).
		Set(bqi.CancelReason.Set(), entity.CancelReason).
		Set(bqi.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update billing queue item: %w", err)
	}

	if err := dberror.CheckRowsAffected(result, "Billing queue item", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) ExistsByShipmentAndType(
	ctx context.Context,
	ti pagination.TenantInfo,
	shipmentID pulid.ID,
	billType billingqueue.BillType,
) (bool, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	return db.NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		Where(bqi.ShipmentID.Eq(), shipmentID).
		Where(bqi.BillType.Eq(), billType).
		Apply(buncolgen.BillingQueueItemApplyTenant(ti)).
		Exists(ctx)
}

func (r *repository) GetStatusCounts(
	ctx context.Context,
	req *repositories.GetBillingQueueStatsRequest,
) (map[billingqueue.Status]int, error) {
	bqi := buncolgen.BillingQueueItemColumns
	db := r.db.DBForContext(ctx)

	var rows []struct {
		Status billingqueue.Status `bun:"status"`
		Count  int                 `bun:"count"`
	}

	if err := db.NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		ColumnExpr(bqi.Status.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Apply(buncolgen.BillingQueueItemApplyTenant(req.TenantInfo)).
		GroupExpr(bqi.Status.Qualified()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	counts := make(map[billingqueue.Status]int, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}

	return counts, nil
}

func tenantInfo(entity *billingqueue.BillingQueueItem) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
