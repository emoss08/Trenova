package accountingconnectionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionapps"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionauth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectioncert"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionhealth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionstatus"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionsubscriptions"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionwebhook"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Service struct {
	health   *accountingconnectionhealth.Service
	subs     *accountingconnectionsubscriptions.Service
	status   *accountingconnectionstatus.Service
	apps     *accountingconnectionapps.Service
	auth     *accountingconnectionauth.Service
	certs    *accountingconnectioncert.Service
	webhooks *accountingconnectionwebhook.Service
}

var _ accountingconnectionports.AccountingConnectionService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	l := p.Logger.Named("service.accounting-connection")
	kit := accountingconnectionkit.New(accountingconnectionkit.Deps{
		Logger:       l,
		DB:           p.DB,
		Connections:  p.Connections,
		Apps:         p.Apps,
		Integrations: p.Integrations,
		Connectors:   p.Connectors,
		Encryption:   p.Encryption,
		Audit:        p.AuditService,
		Realtime:     p.Realtime,
		Watchtower:   p.Watchtower,
		Publisher:    p.Publisher,
		Refresher:    p.Refresher,
		Plans:        p.Plans,
		Certificates: p.Certificates,
	})
	health := accountingconnectionhealth.New(accountingconnectionhealth.Deps{
		Logger:      l,
		DB:          p.DB,
		Connections: p.Connections,
		Kit:         kit,
	})
	subs := accountingconnectionsubscriptions.New(accountingconnectionsubscriptions.Deps{
		Logger:        l,
		Subscriptions: p.Subscriptions,
		Encryption:    p.Encryption,
		Kit:           kit,
		Health:        health,
	})
	status := accountingconnectionstatus.New(
		accountingconnectionstatus.Deps{Connections: p.Connections, Kit: kit, Subs: subs},
	)
	auth := accountingconnectionauth.New(accountingconnectionauth.Deps{
		Logger:       l,
		DB:           p.DB,
		Connections:  p.Connections,
		States:       p.States,
		Certificates: p.Certificates,
		Encryption:   p.Encryption,
		Kit:          kit,
		Subs:         subs,
	})
	return &Service{
		health: health,
		subs:   subs,
		status: status,
		apps: accountingconnectionapps.New(accountingconnectionapps.Deps{
			Logger:      l,
			DB:          p.DB,
			Apps:        p.Apps,
			Connections: p.Connections,
			Encryption:  p.Encryption,
			Audit:       p.AuditService,
			Realtime:    p.Realtime,
			Kit:         kit,
			Status:      status,
		}),
		auth: auth,
		certs: accountingconnectioncert.New(accountingconnectioncert.Deps{
			Logger:        l,
			DB:            p.DB,
			Connections:   p.Connections,
			Certificates:  p.Certificates,
			Encryption:    p.Encryption,
			Organizations: p.Organizations,
			Kit:           kit,
			Auth:          auth,
		}),
		webhooks: accountingconnectionwebhook.New(accountingconnectionwebhook.Deps{
			Logger:        l,
			Apps:          p.Apps,
			Connections:   p.Connections,
			Subscriptions: p.Subscriptions,
			Encryption:    p.Encryption,
			Poller:        p.Poller,
			Kit:           kit,
		}),
	}
}

func (s *Service) Status(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	return s.status.Status(ctx, tenantInfo, typ)
}

func (s *Service) StartAuthorization(
	ctx context.Context,
	req *accountingconnectionports.StartAccountingAuthorizationRequest,
) (*accountingconnectionports.AccountingAuthorizationStart, error) {
	return s.auth.StartAuthorization(ctx, req)
}

func (s *Service) CompleteAuthorization(
	ctx context.Context,
	req *accountingconnectionports.CompleteAccountingAuthorizationRequest,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	return s.auth.CompleteAuthorization(ctx, req)
}

func (s *Service) ChooseCompany(
	ctx context.Context,
	req *accountingconnectionports.ChooseAccountingCompanyRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.auth.ChooseCompany(ctx, req)
}

func (s *Service) Disconnect(
	ctx context.Context,
	req *accountingconnectionports.DisconnectAccountingRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.auth.Disconnect(ctx, req)
}

func (s *Service) PrepareCertificate(
	ctx context.Context,
	req *accountingconnectionports.PrepareAccountingCertificateRequest,
) (*accountingconnectionports.AccountingPreparedCertificate, error) {
	return s.certs.PrepareCertificate(ctx, req)
}

func (s *Service) ConnectWithCertificate(
	ctx context.Context,
	req *accountingconnectionports.ConnectAccountingWithCertificateRequest,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	return s.certs.ConnectWithCertificate(ctx, req)
}

func (s *Service) RenewCertificate(
	ctx context.Context,
	req *accountingconnectionports.RenewAccountingCertificateRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.certs.RenewCertificate(ctx, req)
}

func (s *Service) CheckHealth(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingconnection.AccountingConnection, error) {
	return s.health.CheckHealth(ctx, tenantInfo, connectionID)
}

func (s *Service) CheckDue(
	ctx context.Context,
	limit int,
) (*accountingconnectionports.AccountingHealthSweep, error) {
	return s.health.CheckDue(ctx, limit)
}

func (s *Service) Session(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingconnectionports.AccountingSession, error) {
	return s.health.Session(ctx, tenantInfo, connectionID)
}

func (s *Service) ReportCallFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	cause error,
) accountingsync.ErrorCategory {
	return s.health.ReportCallFailure(ctx, tenantInfo, connectionID, cause)
}

func (s *Service) SyncWebhookSubscriptions(
	ctx context.Context,
	req *accountingconnectionports.SyncAccountingWebhookSubscriptionsRequest,
) (*accountingconnectionports.AccountingSubscriptionSweep, error) {
	return s.subs.SyncWebhookSubscriptions(ctx, req)
}

func (s *Service) ReceiveWebhook(
	ctx context.Context,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) error {
	return s.webhooks.ReceiveWebhook(ctx, req)
}

func (s *Service) SaveApp(
	ctx context.Context,
	req *accountingconnectionports.SaveAccountingAppRequest,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	return s.apps.SaveApp(ctx, req)
}

func (s *Service) RemoveApp(
	ctx context.Context,
	req *accountingconnectionports.RemoveAccountingAppRequest,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	return s.apps.RemoveApp(ctx, req)
}
