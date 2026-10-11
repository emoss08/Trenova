//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package tcasubscriptionrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/tablechangealertcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) tablechangealertports.TCASubscriptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tca-subscription-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *tablechangealertports.ListTCASubscriptionsRequest,
) *bun.SelectQuery {
	cols := tablechangealertcol.TCASubscriptionColumns
	q = querybuilder.ApplyFilters(
		q,
		tablechangealertcol.TCASubscriptionTable.Alias,
		req.Filter,
		(*tablechangealert.TCASubscription)(nil),
	)

	q = q.Apply(tablechangealertcol.TCASubscriptionApplyTenant(req.Filter.TenantInfo)).
		Where(cols.UserID.Eq(), req.Filter.TenantInfo.UserID)

	q = q.Order(cols.CreatedAt.OrderDesc())

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *repository) List(
	ctx context.Context,
	req *tablechangealertports.ListTCASubscriptionsRequest,
) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
			log := r.l.With(zap.String("operation", "List"))

			entities := make(
				[]*tablechangealert.TCASubscription,
				0,
				req.Filter.Pagination.SafeLimit(),
			)
			total, err := r.db.DBForContext(ctx).
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
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *tablechangealertports.ListTCASubscriptionConnectionRequest,
) (*bun.SelectQuery, error) {
	q, err := querybuilder.ApplyCursorFilters(
		q,
		tablechangealertcol.TCASubscriptionTable.Alias,
		req.Filter,
		req.Cursor,
		(*tablechangealert.TCASubscription)(nil),
	)
	if err != nil {
		return nil, err
	}

	return q.Where(
		tablechangealertcol.TCASubscriptionColumns.UserID.Eq(),
		req.Filter.TenantInfo.UserID,
	), nil
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *tablechangealertports.ListTCASubscriptionConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		tablechangealertcol.TCASubscriptionTable.Alias,
		req.Filter,
		(*tablechangealert.TCASubscription)(nil),
	)

	return q.Where(
		tablechangealertcol.TCASubscriptionColumns.UserID.Eq(),
		req.Filter.TenantInfo.UserID,
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *tablechangealertports.ListTCASubscriptionConnectionRequest,
) (*pagination.CursorListResult[*tablechangealert.TCASubscription], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tablechangealert.TCASubscription], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*tablechangealert.TCASubscription)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count tca subscriptions", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*tablechangealert.TCASubscription]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*tablechangealert.TCASubscription) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							ColumnExpr(tablechangealertcol.TCASubscriptionTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return r.applyCursorPageFilters(sq, req)
					},
				})
			if err != nil {
				log.Error("failed to scan tca subscriptions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			log := r.l.With(zap.String("operation", "Create"))

			_, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx)
			if err != nil {
				log.Error("failed to create tca subscription", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			log := r.l.With(
				zap.String("operation", "Update"),
				zap.String("id", entity.ID.String()),
			)

			ov := entity.Version
			entity.Version++
			cols := tablechangealertcol.TCASubscriptionColumns

			results, err := r.db.DBForContext(ctx).
				NewUpdate().
				Model(entity).
				WherePK().
				Where(cols.Version.Eq(), ov).
				Where(cols.UserID.Eq(), entity.UserID).
				OmitZero().
				Returning("*").
				Exec(ctx)
			if err != nil {
				log.Error("failed to update tca subscription", zap.Error(err))
				return nil, err
			}

			if err = dberror.CheckRowsAffected(
				results,
				"TCASubscription",
				entity.ID.String(),
			); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req tablechangealertports.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			log := r.l.With(
				zap.String("operation", "GetByID"),
				zap.String("subscriptionID", req.SubscriptionID.String()),
			)

			entity := new(tablechangealert.TCASubscription)
			cols := tablechangealertcol.TCASubscriptionColumns
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return tablechangealertcol.TCASubscriptionScopeTenant(sq, req.TenantInfo).
						Where(cols.ID.Eq(), req.SubscriptionID).
						Where(cols.UserID.Eq(), req.TenantInfo.UserID)
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get tca subscription", zap.Error(err))
				return nil, dberror.HandleNotFoundError(err, "TCASubscription")
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		log := r.l.With(
			zap.String("operation", "Delete"),
			zap.String("id", id.String()),
		)

		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tablechangealert.TCASubscription)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return tablechangealertcol.TCASubscriptionScopeTenantDelete(dq, tenantInfo).
					Where(tablechangealertcol.TCASubscriptionColumns.ID.Eq(), id).
					Where(tablechangealertcol.TCASubscriptionColumns.UserID.Eq(), tenantInfo.UserID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to delete tca subscription", zap.Error(err))
			return err
		}

		return dberror.CheckRowsAffected(result, "TCASubscription", id.String())
	})
}

func (r *repository) FindMatchingSubscriptions(
	ctx context.Context,
	req tablechangealertports.FindMatchingTCASubscriptionsRequest,
) ([]*tablechangealert.TCASubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tablechangealert.TCASubscription, error) {
			log := r.l.With(
				zap.String("operation", "FindMatchingSubscriptions"),
				zap.String("tableName", req.TableName),
				zap.String("operation_type", req.Operation),
			)

			if !tablechangealert.ValidEventType(req.Operation) {
				return nil, nil
			}

			entities := make([]*tablechangealert.TCASubscription, 0)
			cols := tablechangealertcol.TCASubscriptionColumns
			q := r.db.DBForContext(ctx).
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
		},
	)
}
