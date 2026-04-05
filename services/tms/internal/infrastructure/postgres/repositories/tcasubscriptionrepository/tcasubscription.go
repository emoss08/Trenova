package tcasubscriptionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
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
	cols := buncolgen.TCASubscriptionColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.TCASubscriptionTable.Alias,
		req.Filter,
		(*tablechangealert.TCASubscription)(nil),
	)

	q = q.Apply(buncolgen.TCASubscriptionApplyTenant(req.Filter.TenantInfo)).
		Where(cols.UserID.Eq(), req.Filter.TenantInfo.UserID)

	q = q.Order(cols.CreatedAt.OrderDesc())

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTCASubscriptionsRequest,
) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*tablechangealert.TCASubscription, 0, req.Filter.Pagination.SafeLimit())
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
	cols := buncolgen.TCASubscriptionColumns

	_, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
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
	cols := buncolgen.TCASubscriptionColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TCASubscriptionScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.SubscriptionID).
				Where(cols.UserID.Eq(), req.TenantInfo.UserID)
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
			return buncolgen.TCASubscriptionScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.TCASubscriptionColumns.ID.Eq(), id).
				Where(buncolgen.TCASubscriptionColumns.UserID.Eq(), tenantInfo.UserID)
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
	cols := buncolgen.TCASubscriptionColumns
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(cols.OrganizationID.Eq(), req.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), req.BusinessUnitID).
		Where(cols.TableName.Eq(), req.TableName).
		Where(cols.Status.Eq(), tablechangealert.SubscriptionStatusActive).
		Where(cols.EventTypes.Expr("{} @> ?::jsonb"), `["`+req.Operation+`"]`)

	if req.RecordID != "" {
		q = q.Where(
			buncolgen.Expr("({0} IS NULL OR {0} = '' OR {0} = ?)", cols.RecordID),
			req.RecordID,
		)
	} else {
		q = q.Where(buncolgen.Expr("({} IS NULL OR {} = '')", cols.RecordID))
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to find matching subscriptions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
