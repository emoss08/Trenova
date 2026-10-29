package accountingconnectionsubscriptions

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionhealth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository `optional:"true"`
	Encryption    *encryptionservice.Service
	Kit           *accountingconnectionkit.Kit
	Health        *accountingconnectionhealth.Service
}

type Service struct {
	l             *zap.Logger
	subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository
	encryption    *encryptionservice.Service
	kit           *accountingconnectionkit.Kit
	health        *accountingconnectionhealth.Service
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.accounting-connection"),
		subscriptions: p.Subscriptions,
		encryption:    p.Encryption,
		kit:           p.Kit,
		health:        p.Health,
	}
}

var _ accountingconnectionports.AccountingWebhookSubscriptionSyncer = (*Service)(nil)

func (s *Service) SyncWebhookSubscriptions(
	ctx context.Context,
	req *accountingconnectionports.SyncAccountingWebhookSubscriptionsRequest,
) (*accountingconnectionports.AccountingSubscriptionSweep, error) {
	sweep := &accountingconnectionports.AccountingSubscriptionSweep{}
	if s.subscriptions == nil {
		return sweep, nil
	}
	limit := req.Limit
	if limit <= 0 {
		limit = subscriptionSweepPageSize
	}
	conns, err := s.subscriptions.ListConnections(
		ctx,
		accountingconnectionports.ListWebhookSubscriptionConnectionsRequest{
			IntegrationTypes: accountingprovider.WebhookSubscriptionSystems(),
			AfterID:          req.AfterID,
			Limit:            limit,
		},
	)
	if err != nil {
		return sweep, err
	}
	sweep.Listed = len(conns)
	for _, conn := range conns {
		if ctx.Err() != nil {
			return sweep, ctx.Err()
		}
		sweep.NextAfterID = conn.ID
		if keepErr := s.keepSubscriptions(
			accountingconnectionkit.ConnectionScope(ctx, conn),
			conn,
			sweep,
		); keepErr != nil {
			sweep.Failed++
			s.l.Warn("could not keep accounting webhook subscriptions",
				zap.String("connectionId", conn.ID.String()), zap.Error(keepErr))
		}
	}
	return sweep, nil
}

func (s *Service) keepSubscriptions(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	sweep *accountingconnectionports.AccountingSubscriptionSweep,
) error {
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	rows, err := s.subscriptions.ListByConnection(
		ctx,
		accountingconnectionports.ListAccountingWebhookSubscriptionsRequest{
			TenantInfo:   tenant,
			ConnectionID: conn.ID,
		},
	)
	if err != nil {
		return err
	}
	if !conn.ReadsChanges() {
		return s.dropSubscriptions(ctx, conn, rows, sweep)
	}

	sess, err := s.health.Session(ctx, tenant, conn.ID)
	if err != nil {
		return err
	}
	subscriber, ok := sess.Connector.(accountingconnectorports.AccountingWebhookSubscriber)
	if !ok {
		return nil
	}
	provider, err := s.kit.Provider(conn.IntegrationType)
	if err != nil {
		return err
	}
	url, err := s.notificationURL(ctx, sess.Connection, provider)
	if err != nil {
		return err
	}

	keeper := &subscriptionKeeper{
		svc:        s,
		conn:       sess.Connection,
		tenant:     tenant,
		subscriber: subscriber,
		auth:       accountingconnectorports.DocumentAuthFor(sess.Connection, sess.AccessToken),
		url:        url,
		now:        timeutils.NowUnix(),
		sweep:      sweep,
	}
	return keeper.keep(ctx, rows)
}

func (s *Service) notificationURL(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	provider accountingconnectorports.AccountingProvider,
) (string, error) {
	base, ok := notificationBase(provider)
	if !ok {
		return "", nil
	}
	profile := accountingprovider.MustProfile(conn.IntegrationType)
	if conn.AppSource != accountingconnection.AppSourceTenant {
		return base + profile.WebhookPath(), nil
	}
	cred, found, err := s.kit.TenantCredential(
		ctx,
		pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
		conn.IntegrationType,
	)
	if err != nil {
		return "", err
	}
	if !found {
		return "", accountingconnectionkit.ErrAppChanged
	}
	return base + profile.AppWebhookPath(cred.ID.String()), nil
}

func (s *Service) dropSubscriptions(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	rows []*accountingconnection.AccountingWebhookSubscription,
	sweep *accountingconnectionports.AccountingSubscriptionSweep,
) error {
	if len(rows) == 0 {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	keeper := &subscriptionKeeper{svc: s, conn: conn, tenant: tenant, sweep: sweep}
	if conn.IsActive() {
		if sess, err := s.health.Session(ctx, tenant, conn.ID); err == nil {
			if subscriber, ok := sess.Connector.(accountingconnectorports.AccountingWebhookSubscriber); ok {
				keeper.subscriber = subscriber
				keeper.auth = accountingconnectorports.DocumentAuthFor(
					sess.Connection,
					sess.AccessToken,
				)
			}
		}
	}

	var errs []error
	for _, row := range rows {
		if keeper.subscriber == nil {
			row.Forget()
		}
		if err := keeper.remove(ctx, row); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Service) Release(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if s.subscriptions == nil ||
		!accountingprovider.MustProfile(conn.IntegrationType).WebhookSubscriptions {
		return
	}
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	rows, err := s.subscriptions.ListByConnection(
		ctx,
		accountingconnectionports.ListAccountingWebhookSubscriptionsRequest{
			TenantInfo:   tenant,
			ConnectionID: conn.ID,
		},
	)
	if err == nil {
		err = s.dropSubscriptions(
			ctx,
			conn,
			rows,
			&accountingconnectionports.AccountingSubscriptionSweep{},
		)
	}
	if err != nil {
		s.l.Warn("could not release accounting webhook subscriptions; the renewal job will",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
	}
}

func (s *Service) Summary(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	provider accountingconnectorports.AccountingProvider,
) (*accountingconnectionports.AccountingWebhookSubscriptionSummary, error) {
	if s.subscriptions == nil ||
		!accountingprovider.MustProfile(conn.IntegrationType).WebhookSubscriptions {
		return nil, nil //nolint:nilnil // the provider pushes its webhooks, so there is nothing to summarize
	}
	rows, err := s.subscriptions.ListByConnection(
		ctx,
		accountingconnectionports.ListAccountingWebhookSubscriptionsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: conn.OrganizationID,
				BuID:  conn.BusinessUnitID,
			},
			ConnectionID: conn.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	_, configured := notificationBase(provider)
	summary := &accountingconnectionports.AccountingWebhookSubscriptionSummary{
		Configured: configured,
	}
	for _, row := range rows {
		switch row.Status {
		case accountingconnection.WebhookSubscriptionActive:
			summary.Active++
			if row.ExpiresAt != nil &&
				(summary.NextExpiryAt == nil || *row.ExpiresAt < *summary.NextExpiryAt) {
				expires := *row.ExpiresAt
				summary.NextExpiryAt = &expires
			}
		case accountingconnection.WebhookSubscriptionFailed:
			summary.Failed++
			if summary.LastError == "" {
				summary.LastError = row.LastError
			}
		case accountingconnection.WebhookSubscriptionPending:
			summary.Pending++
		}
	}
	return summary, nil
}
