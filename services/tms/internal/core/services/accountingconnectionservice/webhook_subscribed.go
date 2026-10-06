package accountingconnectionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type subscribedConnection struct {
	tenant pagination.TenantInfo
	id     pulid.ID
}

func (s *Service) receiveSubscribedWebhook(
	ctx context.Context,
	provider services.AccountingProvider,
	req *services.ReceiveAccountingWebhookRequest,
) error {
	subscribing, ok := provider.(services.AccountingSubscriptionProvider)
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

	owned := make([]*accountingsync.AccountingConnection, 0, len(verified))
	realms := make([]string, 0, len(verified))
	for _, target := range verified {
		conn, getErr := s.connections.GetByID(
			dbscope.WithTenant(ctx, target.tenant.DBTenant()),
			repositories.GetAccountingConnectionByIDRequest{TenantInfo: target.tenant, ID: target.id},
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
	provider services.AccountingProvider,
	req *services.ReceiveAccountingWebhookRequest,
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
	req *services.ReceiveAccountingWebhookRequest,
	notifications []services.AccountingWebhookNotification,
) ([]subscribedConnection, error) {
	ids := subscriptionIDs(notifications)
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.subscriptions.ListByExternalIDs(ctx, req.IntegrationType, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*accountingsync.AccountingWebhookSubscription, len(rows))
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
	row *accountingsync.AccountingWebhookSubscription,
	received string,
) bool {
	held, err := s.encryption.DecryptStringWithAAD(row.ClientStateCiphertext, clientStateAAD(row))
	if err != nil {
		s.l.Warn("could not open a webhook subscription's client state",
			zap.String("subscriptionId", row.ID.String()), zap.Error(err))
		return false
	}
	return accountingsync.ClientStatesMatch(held, received)
}
