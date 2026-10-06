package bcconnector

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
)

var _ services.AccountingWebhookSubscriber = (*Connector)(nil)

const subscriptionRenewWindow = 24 * time.Hour

var errUnknownResource = errors.New(
	"businesscentral: the subscription resource is not one Trenova subscribes to",
)

var subscriptionResources = []string{
	"salesInvoices",
	"salesCreditMemos",
	"purchaseInvoices",
	"customers",
	"vendors",
	"items",
	"accounts",
}

func (c *Connector) SubscriptionResources() []string {
	return append([]string(nil), subscriptionResources...)
}

func (c *Connector) SubscriptionRenewWindow() time.Duration {
	return subscriptionRenewWindow
}

func subscriptionInput(
	client *businesscentral.Client,
	resource, notificationURL, clientState string,
) (*businesscentral.SubscriptionInput, error) {
	path := businesscentral.SubscriptionResource(client.Ref(), strings.TrimSpace(resource))
	if path == "" {
		return nil, errUnknownResource
	}
	return &businesscentral.SubscriptionInput{
		NotificationURL: notificationURL,
		Resource:        path,
		ClientState:     clientState,
	}, nil
}

func (c *Connector) Subscribe(
	ctx context.Context,
	req *services.AccountingSubscribeRequest,
) (*accountingsync.WebhookSubscriptionGrant, error) {
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	input, err := subscriptionInput(client, req.Resource, req.NotificationURL, req.ClientState)
	if err != nil {
		return nil, err
	}
	created, err := client.CreateSubscription(ctx, input)
	if err != nil {
		return nil, err
	}
	return c.grantOf(created, input.NotificationURL), nil
}

func (c *Connector) RenewSubscription(
	ctx context.Context,
	req *services.AccountingRenewSubscriptionRequest,
) (*accountingsync.WebhookSubscriptionGrant, error) {
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	input, err := subscriptionInput(client, req.Resource, req.NotificationURL, req.ClientState)
	if err != nil {
		return nil, err
	}
	renewed, err := client.RenewSubscription(ctx, req.ExternalID, req.ETag, input)
	if err != nil {
		return nil, err
	}
	return c.grantOf(renewed, input.NotificationURL), nil
}

func (c *Connector) Unsubscribe(
	ctx context.Context,
	req *services.AccountingUnsubscribeRequest,
) error {
	client, err := c.client(req.Auth)
	if err != nil {
		return err
	}
	if err = client.DeleteSubscription(ctx, req.ExternalID, req.ETag); err != nil &&
		!businesscentral.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *Connector) IsSubscriptionGone(err error) bool {
	return businesscentral.IsNotFound(err)
}

func (c *Connector) grantOf(
	sub *businesscentral.Subscription,
	notificationURL string,
) *accountingsync.WebhookSubscriptionGrant {
	grant := &accountingsync.WebhookSubscriptionGrant{
		ExternalID:      sub.ID,
		NotificationURL: sub.NotificationURL,
		ETag:            sub.ETag,
		ExpiresAt:       sub.ExpirationDateTime,
	}
	if grant.NotificationURL == "" {
		grant.NotificationURL = notificationURL
	}
	if grant.ExpiresAt.IsZero() {
		grant.ExpiresAt = c.now().Add(businesscentral.SubscriptionLifetime)
	}
	return grant
}
