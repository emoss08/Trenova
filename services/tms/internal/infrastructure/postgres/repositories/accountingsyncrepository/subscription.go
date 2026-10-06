package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	subscriptionEntity           = "Accounting webhook subscription"
	defaultSubscriptionPageLimit = 100
	maxSubscriptionLookup        = 1000
)

type SubscriptionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type subscriptionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSubscriptionRepository(
	p SubscriptionParams,
) repositories.AccountingWebhookSubscriptionRepository {
	return &subscriptionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-webhook-subscription-repository"),
	}
}

func (r *subscriptionRepository) ListByConnection(
	ctx context.Context,
	req repositories.ListAccountingWebhookSubscriptionsRequest,
) ([]*accountingsync.AccountingWebhookSubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingWebhookSubscription, error) {
			entities := make([]*accountingsync.AccountingWebhookSubscription, 0)
			cols := buncolgen.AccountingWebhookSubscriptionColumns

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(buncolgen.AccountingWebhookSubscriptionApplyTenant(req.TenantInfo)).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Order(cols.Resource.OrderAsc()).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *subscriptionRepository) ListByExternalIDs(
	ctx context.Context,
	integrationType integration.Type,
	externalIDs []string,
) ([]*accountingsync.AccountingWebhookSubscription, error) {
	if len(externalIDs) == 0 {
		return []*accountingsync.AccountingWebhookSubscription{}, nil
	}
	if len(externalIDs) > maxSubscriptionLookup {
		externalIDs = externalIDs[:maxSubscriptionLookup]
	}
	ctx = dbscope.WithSystem(
		ctx,
		"find the webhook subscriptions a notification names before its tenant is known",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingWebhookSubscription, error) {
			entities := make([]*accountingsync.AccountingWebhookSubscription, 0, len(externalIDs))
			cols := buncolgen.AccountingWebhookSubscriptionColumns

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Where(cols.IntegrationType.Eq(), integrationType).
				Where(cols.ExternalSubscriptionID.In(), bun.List(externalIDs)).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *subscriptionRepository) ListConnections(
	ctx context.Context,
	req repositories.ListWebhookSubscriptionConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	if len(req.IntegrationTypes) == 0 {
		return []*accountingsync.AccountingConnection{}, nil
	}
	ctx = dbscope.WithSystem(
		ctx,
		"list accounting connections whose webhook subscriptions need keeping across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingConnection, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultSubscriptionPageLimit
			}

			db := r.db.DBForContext(ctx)
			entities := make([]*accountingsync.AccountingConnection, 0, limit)
			cols := buncolgen.AccountingConnectionColumns
			subCols := buncolgen.AccountingWebhookSubscriptionColumns

			held := db.NewSelect().
				Model((*accountingsync.AccountingWebhookSubscription)(nil)).
				ColumnExpr("1").
				Where(subCols.ConnectionID.EqColumn(cols.ID)).
				Where(subCols.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
				Where(subCols.OrganizationID.EqColumn(cols.OrganizationID))

			q := db.NewSelect().
				Model(&entities).
				ExcludeColumn(tokenColumns()...).
				Where(cols.IntegrationType.In(), bun.List(req.IntegrationTypes)).
				WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where(cols.Status.NotEq(), accountingsync.ConnectionStatusDisconnected).
						WhereOr("EXISTS (?)", held)
				})
			if !req.AfterID.IsNil() {
				q = q.Where(cols.ID.Gt(), req.AfterID)
			}
			if err := q.Order(cols.ID.OrderAsc()).Limit(limit).Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *subscriptionRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingWebhookSubscription,
) (*accountingsync.AccountingWebhookSubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingWebhookSubscription, error) {
			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *subscriptionRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingWebhookSubscription,
) (*accountingsync.AccountingWebhookSubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingWebhookSubscription, error) {
			cols := buncolgen.AccountingWebhookSubscriptionColumns
			ov := entity.Version
			entity.Version++

			results, err := r.db.DBForContext(ctx).
				NewUpdate().
				Model(entity).
				WherePK().
				Where(cols.Version.Eq(), ov).
				Exec(ctx)
			if err != nil {
				entity.Version = ov
				return nil, err
			}
			if err = dberror.CheckRowsAffected(
				results,
				subscriptionEntity,
				entity.ID.String(),
			); err != nil {
				entity.Version = ov
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *subscriptionRepository) Delete(
	ctx context.Context,
	req repositories.DeleteAccountingWebhookSubscriptionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.AccountingWebhookSubscriptionColumns

		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*accountingsync.AccountingWebhookSubscription)(nil)).
			WhereGroup(" AND ", func(q *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.AccountingWebhookSubscriptionScopeTenantDelete(q, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		return err
	})
}
