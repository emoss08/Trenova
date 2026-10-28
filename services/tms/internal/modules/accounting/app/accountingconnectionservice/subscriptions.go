package accountingconnectionservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const (
	clientStateBytes          = 32
	clientStateField          = "client_state"
	subscriptionSweepPageSize = 50
	failedSubscriptionBackoff = time.Hour
)

var errNoNotificationURL = errors.New(
	"no public https address is configured for webhook notifications, " +
		"so changes are read every five minutes instead",
)

type subscriptionKeeper struct {
	svc        *Service
	conn       *accountingconnection.AccountingConnection
	tenant     pagination.TenantInfo
	subscriber accountingconnectorports.AccountingWebhookSubscriber
	auth       accountingconnectorports.AccountingDocumentAuth
	url        string
	now        int64
	sweep      *accountingconnectionports.AccountingSubscriptionSweep
}

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
		if keepErr := s.keepSubscriptions(connectionScope(ctx, conn), conn, sweep); keepErr != nil {
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

	sess, err := s.Session(ctx, tenant, conn.ID)
	if err != nil {
		return err
	}
	subscriber, ok := sess.Connector.(accountingconnectorports.AccountingWebhookSubscriber)
	if !ok {
		return nil
	}
	provider, err := s.provider(conn.IntegrationType)
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
	cred, found, err := s.tenantCredential(
		ctx,
		pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
		conn.IntegrationType,
	)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errAppChanged
	}
	return base + profile.AppWebhookPath(cred.ID.String()), nil
}

func (k *subscriptionKeeper) keep(
	ctx context.Context,
	rows []*accountingconnection.AccountingWebhookSubscription,
) error {
	byResource := make(map[string]*accountingconnection.AccountingWebhookSubscription, len(rows))
	for _, row := range rows {
		byResource[row.Resource] = row
	}

	var errs []error
	for _, resource := range k.subscriber.SubscriptionResources() {
		row, found := byResource[resource]
		delete(byResource, resource)
		if !found {
			created, err := k.svc.subscriptions.Create(
				ctx,
				&accountingconnection.AccountingWebhookSubscription{
					OrganizationID:  k.conn.OrganizationID,
					BusinessUnitID:  k.conn.BusinessUnitID,
					ConnectionID:    k.conn.ID,
					IntegrationType: k.conn.IntegrationType,
					Resource:        resource,
					Status:          accountingconnection.WebhookSubscriptionPending,
				},
			)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			row = created
		}
		if err := k.keepOne(ctx, row); err != nil {
			errs = append(errs, err)
		}
	}
	for _, stale := range byResource {
		if err := k.remove(ctx, stale); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (k *subscriptionKeeper) keepOne(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) error {
	switch {
	case k.url == "":
		return k.unavailable(ctx, row)
	case row.Status == accountingconnection.WebhookSubscriptionFailed && k.backingOff(row):
		return nil
	case !row.Receives():
		return k.subscribe(ctx, row)
	case row.NotificationURL != k.url:
		k.unsubscribeQuietly(ctx, row)
		row.Forget()
		return k.subscribe(ctx, row)
	case row.RenewDue(k.now, k.subscriber.SubscriptionRenewWindow()):
		return k.renew(ctx, row)
	default:
		return nil
	}
}

func (k *subscriptionKeeper) backingOff(
	row *accountingconnection.AccountingWebhookSubscription,
) bool {
	return row.LastAttemptAt != nil &&
		k.now-*row.LastAttemptAt < int64(failedSubscriptionBackoff/time.Second)
}

func (k *subscriptionKeeper) unavailable(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) error {
	if row.IsHeld() {
		k.unsubscribeQuietly(ctx, row)
		row.Forget()
	}
	if row.Status == accountingconnection.WebhookSubscriptionFailed &&
		row.LastError == errNoNotificationURL.Error() {
		return nil
	}
	row.Fail(errNoNotificationURL.Error(), k.now)
	_, err := k.svc.subscriptions.Update(ctx, row)
	return err
}

func (k *subscriptionKeeper) subscribe(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) error {
	if row.IsHeld() {
		k.unsubscribeQuietly(ctx, row)
		row.Forget()
	}
	clientState, err := tokenutils.RandomHex(clientStateBytes)
	if err != nil {
		return err
	}
	sealed, err := k.svc.encryption.EncryptStringWithAAD(clientState, clientStateAAD(row))
	if err != nil {
		return err
	}

	grant, err := k.subscriber.Subscribe(ctx, &accountingconnectorports.AccountingSubscribeRequest{
		Auth:            k.auth,
		Resource:        row.Resource,
		NotificationURL: k.url,
		ClientState:     clientState,
	})
	if err != nil {
		return k.fail(ctx, row, err)
	}
	row.ClientStateCiphertext = sealed
	row.Activate(grant, k.now)
	if _, err = k.svc.subscriptions.Update(ctx, row); err != nil {
		k.unsubscribeQuietly(ctx, row)
		return err
	}
	k.sweep.Created++
	return nil
}

func (k *subscriptionKeeper) renew(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) error {
	clientState, err := k.svc.encryption.DecryptStringWithAAD(
		row.ClientStateCiphertext,
		clientStateAAD(row),
	)
	if err != nil {
		row.Forget()
		return k.subscribe(ctx, row)
	}
	grant, err := k.subscriber.RenewSubscription(
		ctx,
		&accountingconnectorports.AccountingRenewSubscriptionRequest{
			Auth:            k.auth,
			Resource:        row.Resource,
			ExternalID:      row.ExternalSubscriptionID,
			NotificationURL: row.NotificationURL,
			ClientState:     clientState,
			ETag:            row.ETag,
		},
	)
	if err != nil {
		if k.subscriber.IsSubscriptionGone(err) {
			row.Forget()
			return k.subscribe(ctx, row)
		}
		return k.fail(ctx, row, err)
	}
	row.Activate(grant, k.now)
	if _, err = k.svc.subscriptions.Update(ctx, row); err != nil {
		return err
	}
	k.sweep.Renewed++
	return nil
}

func (k *subscriptionKeeper) fail(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
	cause error,
) error {
	row.Fail(cause.Error(), k.now)
	if _, err := k.svc.subscriptions.Update(ctx, row); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func (k *subscriptionKeeper) remove(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) error {
	k.unsubscribeQuietly(ctx, row)
	if err := k.svc.subscriptions.Delete(
		ctx,
		accountingconnectionports.DeleteAccountingWebhookSubscriptionRequest{
			TenantInfo: k.tenant,
			ID:         row.ID,
		},
	); err != nil {
		return err
	}
	k.sweep.Removed++
	return nil
}

func (k *subscriptionKeeper) unsubscribeQuietly(
	ctx context.Context,
	row *accountingconnection.AccountingWebhookSubscription,
) {
	if !row.IsHeld() {
		return
	}
	err := k.subscriber.Unsubscribe(ctx, &accountingconnectorports.AccountingUnsubscribeRequest{
		Auth:       k.auth,
		ExternalID: row.ExternalSubscriptionID,
		ETag:       row.ETag,
	})
	if err != nil && !k.subscriber.IsSubscriptionGone(err) {
		k.svc.l.Warn("could not delete an accounting webhook subscription",
			zap.String("subscriptionId", row.ID.String()), zap.Error(err))
	}
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
		if sess, err := s.Session(ctx, tenant, conn.ID); err == nil {
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

func (s *Service) releaseSubscriptions(
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

func (s *Service) subscriptionSummary(
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

func notificationBase(provider accountingconnectorports.AccountingProvider) (string, bool) {
	subscribing, ok := provider.(accountingconnectorports.AccountingSubscriptionProvider)
	if !ok {
		return "", false
	}
	base := strings.TrimSuffix(strings.TrimSpace(subscribing.NotificationBaseURL()), "/")
	if !strings.HasPrefix(base, "https://") {
		return "", false
	}
	return base, true
}

func clientStateAAD(row *accountingconnection.AccountingWebhookSubscription) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingWebhookState,
		OrganizationID: row.OrganizationID,
		BusinessUnitID: row.BusinessUnitID,
		ResourceID:     row.ID.String() + ":" + clientStateField,
	}
}

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
