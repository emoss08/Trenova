package bcconnector

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralwebhook"
)

var _ accountingconnectorports.AccountingWebhookSubscriber = (*Connector)(nil)

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
) (*businesscentralwebhook.SubscriptionInput, error) {
	path := businesscentralwebhook.SubscriptionResource(client.Ref(), strings.TrimSpace(resource))
	if path == "" {
		return nil, errUnknownResource
	}
	return &businesscentralwebhook.SubscriptionInput{
		NotificationURL: notificationURL,
		Resource:        path,
		ClientState:     clientState,
	}, nil
}

func (c *Connector) Subscribe(
	ctx context.Context,
	req *accountingconnectorports.AccountingSubscribeRequest,
) (*accountingconnection.WebhookSubscriptionGrant, error) {
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	input, err := subscriptionInput(client, req.Resource, req.NotificationURL, req.ClientState)
	if err != nil {
		return nil, err
	}
	created, err := businesscentralwebhook.CreateSubscription(ctx, client, input)
	if err != nil {
		return nil, err
	}
	return c.grantOf(created, input.NotificationURL), nil
}

func (c *Connector) RenewSubscription(
	ctx context.Context,
	req *accountingconnectorports.AccountingRenewSubscriptionRequest,
) (*accountingconnection.WebhookSubscriptionGrant, error) {
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	input, err := subscriptionInput(client, req.Resource, req.NotificationURL, req.ClientState)
	if err != nil {
		return nil, err
	}
	renewed, err := businesscentralwebhook.RenewSubscription(
		ctx,
		client,
		req.ExternalID,
		req.ETag,
		input,
	)
	if err != nil {
		return nil, err
	}
	return c.grantOf(renewed, input.NotificationURL), nil
}

func (c *Connector) Unsubscribe(
	ctx context.Context,
	req *accountingconnectorports.AccountingUnsubscribeRequest,
) error {
	client, err := c.client(req.Auth)
	if err != nil {
		return err
	}
	if err = businesscentralwebhook.DeleteSubscription(ctx, client, req.ExternalID, req.ETag); err != nil &&
		!businesscentralerrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *Connector) IsSubscriptionGone(err error) bool {
	return businesscentralerrors.IsNotFound(err)
}

func (c *Connector) grantOf(
	sub *businesscentralwebhook.Subscription,
	notificationURL string,
) *accountingconnection.WebhookSubscriptionGrant {
	grant := &accountingconnection.WebhookSubscriptionGrant{
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
