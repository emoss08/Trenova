package subscriptionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName             = "Subscription"
	defaultListDueLimit    = 500
	maxListDueLimit        = 5_000
	listDueScopeReason     = "list cloud subscriptions due a lifecycle transition across every organization"
	countStatusScopeReason = "count cloud subscriptions by status across every organization"
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

func New(p Params) repositories.SubscriptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.subscription-repository"),
	}
}

func (r *repository) GetByOrganization(
	ctx context.Context,
	req repositories.GetSubscriptionRequest,
) (*subscription.Subscription, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*subscription.Subscription, error) {
		entity := new(subscription.Subscription)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(buncolgen.SubscriptionApplyTenant(req.TenantInfo)).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *subscription.Subscription,
) (*subscription.Subscription, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*subscription.Subscription, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			r.l.Error("failed to create subscription",
				zap.String("organizationId", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateSubscriptionStatusRequest,
) (*subscription.Subscription, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*subscription.Subscription, error) {
		cols := buncolgen.SubscriptionColumns
		entity := new(subscription.Subscription)

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.SubscriptionScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update subscription status",
				zap.String("subscriptionId", req.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckRowsAffected(result, entityName, req.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListDue(
	ctx context.Context,
	req *repositories.ListDueSubscriptionsRequest,
) ([]*subscription.Subscription, error) {
	ctx = dbscope.WithSystem(ctx, listDueScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*subscription.Subscription, error) {
		cols := buncolgen.SubscriptionColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultListDueLimit
		}
		limit = min(limit, maxListDueLimit)

		entities := make([]*subscription.Subscription, 0, min(limit, 64))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					WhereGroup(" OR ", func(trial *bun.SelectQuery) *bun.SelectQuery {
						return trial.
							Where(cols.Status.Eq(), subscription.StatusTrialing).
							Where(cols.TrialEndsAt.Lte(), req.Now)
					}).
					WhereGroup(" OR ", func(grace *bun.SelectQuery) *bun.SelectQuery {
						return grace.
							Where(cols.Status.Eq(), subscription.StatusReadOnly).
							Where(cols.ReadOnlyUntil.Lte(), req.Now)
					})
			}).
			Order(cols.TrialEndsAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list due subscriptions", zap.Error(err))
			return nil, fmt.Errorf("list due subscriptions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) CountByStatus(
	ctx context.Context,
	req *repositories.CountSubscriptionsByStatusRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, countStatusScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.Statuses) == 0 {
			return 0, nil
		}

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*subscription.Subscription)(nil)).
			Where(buncolgen.SubscriptionColumns.Status.In(), bun.List(req.Statuses)).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count subscriptions by status", zap.Error(err))
			return 0, fmt.Errorf("count subscriptions by status: %w", err)
		}

		return count, nil
	})
}
