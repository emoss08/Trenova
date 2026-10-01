package accountingconnectionservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const appCredentialPrefix = "acctapp_"

type webhookSender struct {
	identity accountingsync.AppIdentity
	tenant   *pagination.TenantInfo
}

func (w *webhookSender) owns(conn *accountingsync.AccountingConnection) bool {
	if w.tenant != nil &&
		(conn.OrganizationID != w.tenant.OrgID || conn.BusinessUnitID != w.tenant.BuID) {
		return false
	}
	return conn.ConnectedThrough(w.identity)
}

func errWebhookUnverified() error {
	return errortypes.NewAuthenticationError("The webhook signature did not verify")
}

func (s *Service) ReceiveWebhook(
	ctx context.Context,
	req *services.ReceiveAccountingWebhookRequest,
) error {
	provider, err := s.provider(req.IntegrationType)
	if err != nil {
		return err
	}

	sender, err := s.webhookSender(ctx, provider, req)
	if err != nil {
		return err
	}
	if sender == nil {
		return s.receiveLegacyWebhook(ctx, provider, req)
	}

	realms, err := webhookRealms(provider, req.Body)
	if err != nil {
		return err
	}
	if len(realms) == 0 {
		return nil
	}
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		repositories.ListAccountingConnectionsByRealmRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        realms,
		},
	)
	if err != nil {
		return err
	}

	owned := make([]*accountingsync.AccountingConnection, 0, len(holders))
	verified := make([]string, 0, len(holders))
	for _, holder := range holders {
		if sender.owns(holder) {
			owned = append(owned, holder)
			verified = append(verified, holder.ExternalRealmID)
		}
	}
	if len(verified) == 0 {
		s.l.Debug("verified webhook for companies with no connection through its app",
			zap.Int("realms", len(realms)))
		return nil
	}
	return s.recordWebhook(ctx, req, owned, verified)
}

func (s *Service) webhookSender(
	ctx context.Context,
	provider services.AccountingProvider,
	req *services.ReceiveAccountingWebhookRequest,
) (*webhookSender, error) {
	if req.AppID != "" {
		return s.tenantWebhookSender(ctx, provider, req)
	}

	instance, ok := provider.InstanceApp()
	if !ok {
		return nil, nil //nolint:nilnil // no instance app sends the legacy per-connection check
	}
	connector, err := provider.Bind(instance)
	if err != nil {
		return nil, err
	}
	if verifyErr := connector.VerifyWebhook(req.Signature, req.Body); verifyErr == nil {
		return &webhookSender{identity: instance.Identity()}, nil
	}
	return nil, nil //nolint:nilnil // a tenant app on the shared address is checked per connection
}

func (s *Service) tenantWebhookSender(
	ctx context.Context,
	provider services.AccountingProvider,
	req *services.ReceiveAccountingWebhookRequest,
) (*webhookSender, error) {
	if !pulid.LooksLike(req.AppID) || pulid.ID(req.AppID).Prefix() != appCredentialPrefix {
		return nil, errWebhookUnverified()
	}
	cred, err := s.apps.GetForWebhook(ctx, pulid.ID(req.AppID), req.IntegrationType)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errWebhookUnverified()
		}
		return nil, err
	}
	app, err := s.openCredential(cred)
	if err != nil {
		s.l.Warn("could not open the app named by a webhook",
			zap.String("appId", req.AppID), zap.Error(err))
		return nil, errWebhookUnverified()
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	if connector.VerifyWebhook(req.Signature, req.Body) != nil {
		return nil, errWebhookUnverified()
	}
	return &webhookSender{
		identity: app.Identity(),
		tenant:   &pagination.TenantInfo{OrgID: cred.OrganizationID, BuID: cred.BusinessUnitID},
	}, nil
}

func webhookRealms(provider services.AccountingProvider, body []byte) ([]string, error) {
	realms, err := provider.WebhookRealmIDs(body)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The webhook body could not be read",
		)
	}
	return realms, nil
}

func (s *Service) receiveLegacyWebhook(
	ctx context.Context,
	provider services.AccountingProvider,
	req *services.ReceiveAccountingWebhookRequest,
) error {
	realms, err := webhookRealms(provider, req.Body)
	if err != nil {
		return err
	}
	if len(realms) == 0 {
		return errWebhookUnverified()
	}
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		repositories.ListAccountingConnectionsByRealmRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        realms,
		},
	)
	if err != nil {
		return err
	}

	verified := s.verifiedRealms(ctx, provider, holders, req)
	if len(verified) == 0 {
		return errWebhookUnverified()
	}
	return s.recordWebhook(ctx, req, holders, verified)
}

func (s *Service) recordWebhook(
	ctx context.Context,
	req *services.ReceiveAccountingWebhookRequest,
	holders []*accountingsync.AccountingConnection,
	verified []string,
) error {
	if _, err := s.connections.MarkWebhookReceived(
		ctx,
		repositories.MarkAccountingWebhookRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        verified,
			ReceivedAt:      timeutils.NowUnix(),
		},
	); err != nil {
		return err
	}

	s.pollChanged(ctx, holders, verified)
	return nil
}

func (s *Service) pollChanged(
	ctx context.Context,
	holders []*accountingsync.AccountingConnection,
	verified []string,
) {
	if s.poller == nil {
		return
	}
	for _, holder := range holders {
		if !slices.Contains(verified, holder.ExternalRealmID) || !holder.ReadsChanges() {
			continue
		}
		tenant := pagination.TenantInfo{OrgID: holder.OrganizationID, BuID: holder.BusinessUnitID}
		if err := s.poller.PollNow(connectionScope(ctx, holder), tenant, holder.ID); err != nil {
			s.l.Warn("failed to wake the change reader after a webhook",
				zap.String("connectionId", holder.ID.String()), zap.Error(err))
		}
	}
}

func connectionScope(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
	})
}

func (s *Service) verifiedRealms(
	ctx context.Context,
	provider services.AccountingProvider,
	holders []*accountingsync.AccountingConnection,
	req *services.ReceiveAccountingWebhookRequest,
) []string {
	byApp := make(map[string]bool, len(holders))
	verified := make([]string, 0, len(holders))
	for _, holder := range holders {
		app, err := s.appForConnection(connectionScope(ctx, holder), provider, holder)
		if err != nil {
			continue
		}
		key := string(app.Source) + ":" + app.Identity().Fingerprint
		ok, seen := byApp[key]
		if !seen {
			connector, bindErr := provider.Bind(app)
			ok = bindErr == nil && connector.VerifyWebhook(req.Signature, req.Body) == nil
			byApp[key] = ok
		}
		if ok {
			verified = append(verified, holder.ExternalRealmID)
		}
	}
	return verified
}
