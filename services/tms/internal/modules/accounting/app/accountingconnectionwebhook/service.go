package accountingconnectionwebhook

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func subscriptionIDs(
	notifications []accountingconnectorports.AccountingWebhookNotification,
) []string {
	ids := make([]string, 0, len(notifications))
	seen := make(map[string]struct{}, len(notifications))
	for idx := range notifications {
		id := notifications[idx].SubscriptionID
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Apps          accountingconnectionports.AccountingAppCredentialRepository
	Connections   accountingconnectionports.AccountingConnectionRepository
	Subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository `optional:"true"`
	Encryption    *encryptionservice.Service
	Poller        accountinginboundports.AccountingChangePoller `optional:"true"`
	Kit           *accountingconnectionkit.Kit
}

type Service struct {
	l             *zap.Logger
	apps          accountingconnectionports.AccountingAppCredentialRepository
	connections   accountingconnectionports.AccountingConnectionRepository
	subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository
	encryption    *encryptionservice.Service
	poller        accountinginboundports.AccountingChangePoller
	kit           *accountingconnectionkit.Kit
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.accounting-connection"),
		apps:          p.Apps,
		connections:   p.Connections,
		subscriptions: p.Subscriptions,
		encryption:    p.Encryption,
		poller:        p.Poller,
		kit:           p.Kit,
	}
}

func (s *Service) ReceiveWebhook(
	ctx context.Context,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) error {
	provider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return err
	}
	if accountingprovider.MustProfile(req.IntegrationType).WebhookSubscriptions {
		return s.receiveSubscribedWebhook(ctx, provider, req)
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
		accountingconnectionports.ListAccountingConnectionsByRealmRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        realms,
		},
	)
	if err != nil {
		return err
	}

	owned := make([]*accountingconnection.AccountingConnection, 0, len(holders))
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
	provider accountingconnectorports.AccountingProvider,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
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
	provider accountingconnectorports.AccountingProvider,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) (*webhookSender, error) {
	app, sender, err := s.namedApp(ctx, req)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	if connector.VerifyWebhook(req.Signature, req.Body) != nil {
		return nil, errWebhookUnverified()
	}
	return sender, nil
}

func (s *Service) namedApp(
	ctx context.Context,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) (*accountingconnectorports.AccountingApp, *webhookSender, error) {
	if !pulid.LooksLike(req.AppID) || pulid.ID(req.AppID).Prefix() != appCredentialPrefix {
		return nil, nil, errWebhookUnverified()
	}
	cred, err := s.apps.GetForWebhook(ctx, pulid.ID(req.AppID), req.IntegrationType)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil, errWebhookUnverified()
		}
		return nil, nil, err
	}
	app, err := s.kit.OpenCredential(cred)
	if err != nil {
		s.l.Warn("could not open the app named by a webhook",
			zap.String("appId", req.AppID), zap.Error(err))
		return nil, nil, errWebhookUnverified()
	}
	return app, &webhookSender{
		identity: app.Identity(),
		tenant:   &pagination.TenantInfo{OrgID: cred.OrganizationID, BuID: cred.BusinessUnitID},
	}, nil
}

func (s *Service) receiveLegacyWebhook(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
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
		accountingconnectionports.ListAccountingConnectionsByRealmRequest{
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
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
	holders []*accountingconnection.AccountingConnection,
	verified []string,
) error {
	if _, err := s.connections.MarkWebhookReceived(
		ctx,
		accountingconnectionports.MarkAccountingWebhookRequest{
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
	holders []*accountingconnection.AccountingConnection,
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
		if err := s.poller.PollNow(accountingconnectionkit.ConnectionScope(ctx, holder), tenant, holder.ID); err != nil {
			s.l.Warn("failed to wake the change reader after a webhook",
				zap.String("connectionId", holder.ID.String()), zap.Error(err))
		}
	}
}

func (s *Service) verifiedRealms(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	holders []*accountingconnection.AccountingConnection,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) []string {
	byApp := make(map[string]bool, len(holders))
	verified := make([]string, 0, len(holders))
	for _, holder := range holders {
		app, err := s.kit.AppForConnection(
			accountingconnectionkit.ConnectionScope(ctx, holder),
			provider,
			holder,
		)
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

func (s *Service) receiveSubscribedWebhook(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) error {
	subscribing, ok := provider.(accountingconnectorports.AccountingSubscriptionProvider)
	if !ok || s.subscriptions == nil {
		return errWebhookUnverified()
	}
	sender, err := s.subscribedSender(ctx, provider, req)
	if err != nil {
		return err
	}
	notifications, err := subscribing.ParseNotifications(req.Body)
	if err != nil {
		return errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The webhook body could not be read",
		)
	}

	verified, err := s.verifiedSubscriptions(ctx, req, notifications)
	if err != nil {
		return err
	}
	if len(verified) == 0 {
		return errWebhookUnverified()
	}

	owned := make([]*accountingconnection.AccountingConnection, 0, len(verified))
	realms := make([]string, 0, len(verified))
	for _, target := range verified {
		conn, getErr := s.connections.GetByID(
			dbscope.WithTenant(ctx, target.tenant.DBTenant()),
			accountingconnectionports.GetAccountingConnectionByIDRequest{
				TenantInfo: target.tenant,
				ID:         target.id,
			},
		)
		if getErr != nil {
			if errortypes.IsNotFoundError(getErr) {
				continue
			}
			return getErr
		}
		if !conn.IsActive() || !sender.owns(conn) {
			continue
		}
		owned = append(owned, conn)
		realms = append(realms, conn.ExternalRealmID)
	}
	if len(owned) == 0 {
		return errWebhookUnverified()
	}
	return s.recordWebhook(ctx, req, owned, realms)
}

func (s *Service) subscribedSender(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
) (*webhookSender, error) {
	if req.AppID != "" {
		_, sender, err := s.namedApp(ctx, req)
		return sender, err
	}
	instance, ok := provider.InstanceApp()
	if !ok {
		return nil, errWebhookUnverified()
	}
	return &webhookSender{identity: instance.Identity()}, nil
}

func (s *Service) verifiedSubscriptions(
	ctx context.Context,
	req *accountingconnectionports.ReceiveAccountingWebhookRequest,
	notifications []accountingconnectorports.AccountingWebhookNotification,
) ([]subscribedConnection, error) {
	ids := subscriptionIDs(notifications)
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.subscriptions.ListByExternalIDs(ctx, req.IntegrationType, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*accountingconnection.AccountingWebhookSubscription, len(rows))
	for _, row := range rows {
		byID[row.ExternalSubscriptionID] = row
	}

	seen := make(map[pulid.ID]struct{}, len(rows))
	verified := make([]subscribedConnection, 0, len(rows))
	for idx := range notifications {
		note := &notifications[idx]
		row, found := byID[note.SubscriptionID]
		if !found || !row.Receives() || !s.clientStateMatches(row, note.ClientState) {
			continue
		}
		if _, dup := seen[row.ConnectionID]; dup {
			continue
		}
		seen[row.ConnectionID] = struct{}{}
		verified = append(verified, subscribedConnection{
			tenant: pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
			id:     row.ConnectionID,
		})
	}
	return verified, nil
}

func (s *Service) clientStateMatches(
	row *accountingconnection.AccountingWebhookSubscription,
	received string,
) bool {
	held, err := s.encryption.DecryptStringWithAAD(
		row.ClientStateCiphertext,
		accountingconnectionkit.ClientStateAAD(row),
	)
	if err != nil {
		s.l.Warn("could not open a webhook subscription's client state",
			zap.String("subscriptionId", row.ID.String()), zap.Error(err))
		return false
	}
	return accountingconnection.ClientStatesMatch(held, received)
}

var _ accountingconnectionports.AccountingWebhookReceiver = (*Service)(nil)
