package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection/accountingconnectiondb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	subscriptionEntity           = "Accounting webhook subscription"
	defaultSubscriptionPageLimit = 100
	maxSubscriptionLookup        = 1000
)

var subscriptionErrors = dberror.Mapping{Entity: subscriptionEntity}

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
) accountingconnectionports.AccountingWebhookSubscriptionRepository {
	return &subscriptionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-webhook-subscription-repository"),
	}
}

func (r *subscriptionRepository) ListByConnection(
	ctx context.Context,
	req accountingconnectionports.ListAccountingWebhookSubscriptionsRequest,
) ([]*accountingconnection.AccountingWebhookSubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingconnection.AccountingWebhookSubscription, error) {
			ws := accountingconnectiondb.AccountingWebhookSubscriptions
			return ws.Where(ws.ConnectionID.Eq(req.ConnectionID)).
				Order(ws.Resource.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *subscriptionRepository) ListByExternalIDs(
	ctx context.Context,
	integrationType integration.Type,
	externalIDs []string,
) ([]*accountingconnection.AccountingWebhookSubscription, error) {
	if len(externalIDs) == 0 {
		return []*accountingconnection.AccountingWebhookSubscription{}, nil
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
		func(ctx context.Context) ([]*accountingconnection.AccountingWebhookSubscription, error) {
			ws := accountingconnectiondb.AccountingWebhookSubscriptions
			return ws.AllTenants().
				Where(
					ws.IntegrationType.Eq(integrationType),
					ws.ExternalSubscriptionID.In(externalIDs...),
				).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *subscriptionRepository) ListConnections(
	ctx context.Context,
	req accountingconnectionports.ListWebhookSubscriptionConnectionsRequest,
) ([]*accountingconnection.AccountingConnection, error) {
	if len(req.IntegrationTypes) == 0 {
		return []*accountingconnection.AccountingConnection{}, nil
	}
	ctx = dbscope.WithSystem(
		ctx,
		"list accounting connections whose webhook subscriptions need keeping across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingconnection.AccountingConnection, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultSubscriptionPageLimit
			}

			ac := accountingconnectiondb.AccountingConnections
			ws := accountingconnectiondb.AccountingWebhookSubscriptions
			held := ws.AllTenants().Where(
				ws.ConnectionID.EqCol(ac.ID),
				ws.BusinessUnitID.EqCol(ac.BusinessUnitID),
				ws.OrganizationID.EqCol(ac.OrganizationID),
			)

			return withoutTokens().
				AllTenants().
				Where(
					ac.IntegrationType.In(req.IntegrationTypes...),
					keel.Or(
						ac.Status.Ne(accountingconnection.ConnectionStatusDisconnected),
						keel.Exists(held),
					),
					keel.When(!req.AfterID.IsNil(), ac.ID.Gt(req.AfterID)),
				).
				Order(ac.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *subscriptionRepository) Create(
	ctx context.Context,
	entity *accountingconnection.AccountingWebhookSubscription,
) (*accountingconnection.AccountingWebhookSubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingWebhookSubscription, error) {
			err := accountingconnectiondb.AccountingWebhookSubscriptions.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				return nil, subscriptionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *subscriptionRepository) Update(
	ctx context.Context,
	entity *accountingconnection.AccountingWebhookSubscription,
) (*accountingconnection.AccountingWebhookSubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingWebhookSubscription, error) {
			ov := entity.Version
			err := accountingconnectiondb.AccountingWebhookSubscriptions.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				entity.Version = ov
				return nil, subscriptionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *subscriptionRepository) Delete(
	ctx context.Context,
	req accountingconnectionports.DeleteAccountingWebhookSubscriptionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ws := accountingconnectiondb.AccountingWebhookSubscriptions
		_, err := ws.Where(ws.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		return err
	})
}
