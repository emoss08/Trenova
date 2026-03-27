package tcasubscriptionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.TCASubscriptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tca-subscription-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTCASubscriptionsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"tcas",
		req.Filter,
		(*tablechangealert.TCASubscription)(nil),
	)

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("tcas.organization_id = ?", req.Filter.TenantInfo.OrgID).
			Where("tcas.business_unit_id = ?", req.Filter.TenantInfo.BuID).
			Where("tcas.user_id = ?", req.Filter.TenantInfo.UserID)
	})

	q = q.Order("tcas.created_at DESC")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTCASubscriptionsRequest,
) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*tablechangealert.TCASubscription, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list tca subscriptions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tablechangealert.TCASubscription]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	log := r.l.With(zap.String("operation", "Create"))

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create tca subscription", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	_, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update tca subscription", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("subscriptionID", req.SubscriptionID.String()),
	)

	entity := new(tablechangealert.TCASubscription)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tcas.id = ?", req.SubscriptionID).
				Where("tcas.organization_id = ?", req.TenantInfo.OrgID).
				Where("tcas.business_unit_id = ?", req.TenantInfo.BuID).
				Where("tcas.user_id = ?", req.TenantInfo.UserID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get tca subscription", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "TCASubscription")
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", id.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*tablechangealert.TCASubscription)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("tcas.id = ?", id).
				Where("tcas.organization_id = ?", tenantInfo.OrgID).
				Where("tcas.business_unit_id = ?", tenantInfo.BuID).
				Where("tcas.user_id = ?", tenantInfo.UserID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete tca subscription", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "TCASubscription", id.String())
}

func (r *repository) FindMatchingSubscriptions(
	ctx context.Context,
	req repositories.FindMatchingTCASubscriptionsRequest,
) ([]*tablechangealert.TCASubscription, error) {
	log := r.l.With(
		zap.String("operation", "FindMatchingSubscriptions"),
		zap.String("tableName", req.TableName),
		zap.String("operation_type", req.Operation),
	)

	if !tablechangealert.ValidEventType(req.Operation) {
		return nil, nil
	}

	entities := make([]*tablechangealert.TCASubscription, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where("tcas.organization_id = ?", req.OrganizationID).
		Where("tcas.business_unit_id = ?", req.BusinessUnitID).
		Where("tcas.table_name = ?", req.TableName).
		Where("tcas.status = ?", tablechangealert.SubscriptionStatusActive).
		Where("tcas.event_types @> ?::jsonb", `["`+req.Operation+`"]`)

	if req.RecordID != "" {
		q = q.Where("(tcas.record_id IS NULL OR tcas.record_id = '' OR tcas.record_id = ?)", req.RecordID)
	} else {
		q = q.Where("(tcas.record_id IS NULL OR tcas.record_id = '')")
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to find matching subscriptions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
