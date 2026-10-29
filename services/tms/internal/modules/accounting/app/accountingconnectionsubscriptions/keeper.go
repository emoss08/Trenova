package accountingconnectionsubscriptions

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const (
	clientStateBytes          = 32
	subscriptionSweepPageSize = 50
	failedSubscriptionBackoff = time.Hour
)

var ErrNoNotificationURL = errors.New(
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
		row.LastError == ErrNoNotificationURL.Error() {
		return nil
	}
	row.Fail(ErrNoNotificationURL.Error(), k.now)
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
	sealed, err := k.svc.encryption.EncryptStringWithAAD(
		clientState,
		accountingconnectionkit.ClientStateAAD(row),
	)
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
		accountingconnectionkit.ClientStateAAD(row),
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
