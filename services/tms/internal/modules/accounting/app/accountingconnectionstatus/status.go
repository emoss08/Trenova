package accountingconnectionstatus

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionsubscriptions"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) Status(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	provider, err := s.kit.Provider(typ)
	if err != nil {
		return nil, err
	}
	settings, err := s.kit.AppSettings(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}

	status := &accountingconnectionports.AccountingSyncStatus{
		IntegrationType: typ,
		ProviderName:    accountingprovider.ProviderName(typ),
		Profile:         accountingprovider.MustProfile(typ),
		Available:       settings.ActiveSource != "" && settings.RedirectURL != "",
		App:             settings,
	}

	conn, err := s.connections.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	switch {
	case err == nil:
		status.Connection = conn
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}
	if status.Connection != nil {
		if status.WebhookSubscriptions, err = s.subs.Summary(
			ctx,
			status.Connection,
			provider,
		); err != nil {
			return nil, err
		}
	}
	if status.PendingCertificate, err = s.kit.PendingCertificate(ctx, tenantInfo, typ); err != nil {
		return nil, err
	}

	return status, nil
}

type Deps struct {
	Connections accountingconnectionports.AccountingConnectionRepository
	Kit         *accountingconnectionkit.Kit
	Subs        *accountingconnectionsubscriptions.Service
}

type Service struct {
	connections accountingconnectionports.AccountingConnectionRepository
	kit         *accountingconnectionkit.Kit
	subs        *accountingconnectionsubscriptions.Service
}

func New(d Deps) *Service {
	return &Service{connections: d.Connections, kit: d.Kit, subs: d.Subs}
}
