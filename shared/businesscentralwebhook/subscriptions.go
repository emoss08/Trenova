package businesscentralwebhook

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
)

const (
	subscriptionsEntity     = "subscriptions"
	subscriptionResourceAPI = "/" + businesscentral.APISegment + "/"
	maxSubscriptionIDLength = 100
	maxResourceLength       = 250
	maxNotificationURLSize  = 2048
	httpsScheme             = "https"
)

type Subscription struct {
	ID                 string
	NotificationURL    string
	Resource           string
	Timestamp          int64
	UserID             string
	ClientState        string
	ExpirationDateTime time.Time
	CreatedAt          time.Time
	LastModified       time.Time
	ETag               string
}

type SubscriptionInput struct {
	NotificationURL string
	Resource        string
	ClientState     string
}

type wireSubscription struct {
	SubscriptionID       string                   `json:"subscriptionId"`
	NotificationURL      string                   `json:"notificationUrl"`
	Resource             string                   `json:"resource"`
	Timestamp            int64                    `json:"timestamp"`
	UserID               string                   `json:"userId"`
	ClientState          string                   `json:"clientState"`
	ExpirationDateTime   businesscentral.WireTime `json:"expirationDateTime"`
	SystemCreatedAt      businesscentral.WireTime `json:"systemCreatedAt"`
	LastModifiedDateTime businesscentral.WireTime `json:"lastModifiedDateTime"`
	ETag                 string                   `json:"@odata.etag"`
}

func (w *wireSubscription) subscription() Subscription {
	return Subscription{
		ID:                 w.SubscriptionID,
		NotificationURL:    w.NotificationURL,
		Resource:           w.Resource,
		Timestamp:          w.Timestamp,
		UserID:             businesscentral.EmptyGUID(w.UserID),
		ClientState:        w.ClientState,
		ExpirationDateTime: w.ExpirationDateTime.Time(),
		CreatedAt:          w.SystemCreatedAt.Time(),
		LastModified:       w.LastModifiedDateTime.Time(),
		ETag:               w.ETag,
	}
}

type subscriptionBody struct {
	NotificationURL string `json:"notificationUrl"`
	Resource        string `json:"resource"`
	ClientState     string `json:"clientState,omitempty"`
}

func SubscriptionResource(ref businesscentral.CompanyRef, entity string) string {
	normalized, err := businesscentral.NewCompanyRef(ref.TenantID, ref.Environment, ref.CompanyID)
	if err != nil || !validEntityName(entity) {
		return ""
	}
	return subscriptionResourceAPI + businesscentral.CompaniesEntity + "(" + normalized.CompanyID + ")/" + entity
}

func Subscriptions(ctx context.Context, c *businesscentral.Client) ([]Subscription, error) {
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "subscriptions",
		Path:     c.EnvPath() + "/" + subscriptionsEntity,
		Unpaged:  true,
	}, (*wireSubscription).subscription)
}

func CreateSubscription(
	ctx context.Context,
	c *businesscentral.Client,
	in *SubscriptionInput,
) (*Subscription, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: "subscriptions-create",
		Method:   http.MethodPost,
		Path:     c.EnvPath() + "/" + subscriptionsEntity,
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, (*wireSubscription).subscription)
}

func RenewSubscription(
	ctx context.Context,
	c *businesscentral.Client,
	subscriptionID, etag string,
	in *SubscriptionInput,
) (*Subscription, error) {
	path, err := subscriptionPath(c, subscriptionID)
	if err != nil {
		return nil, err
	}
	match, err := businesscentral.IfMatch(etag)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: "subscriptions-renew",
		Method:   http.MethodPatch,
		Path:     path,
		IfMatch:  match,
		Body:     body,
	}, (*wireSubscription).subscription)
}

func DeleteSubscription(
	ctx context.Context,
	c *businesscentral.Client,
	subscriptionID, etag string,
) error {
	path, err := subscriptionPath(c, subscriptionID)
	if err != nil {
		return err
	}
	return c.Remove(ctx, "subscriptions-delete", path, etag)
}

func subscriptionPath(c *businesscentral.Client, subscriptionID string) (string, error) {
	id := strings.TrimSpace(subscriptionID)
	if !validSubscriptionID(id) {
		return "", businesscentralerrors.ErrInvalidSubscriptionID
	}
	return c.EnvPath() + "/" + subscriptionsEntity + "('" + id + "')", nil
}

func (in *SubscriptionInput) body() (subscriptionBody, error) {
	target := strings.TrimSpace(in.NotificationURL)
	parsed, err := url.Parse(target)
	if err != nil || len(target) > maxNotificationURLSize || parsed.Scheme != httpsScheme ||
		parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return subscriptionBody{}, businesscentralerrors.ErrNotificationURL
	}
	resource := strings.TrimSpace(in.Resource)
	if resource == "" {
		return subscriptionBody{}, businesscentralerrors.ErrResourceRequired
	}
	if !strings.HasPrefix(resource, subscriptionResourceAPI) ||
		len(resource) > maxResourceLength || businesscentral.HasControl(resource) ||
		strings.ContainsAny(resource, " ?#") {
		return subscriptionBody{}, businesscentralerrors.ErrResourceRequired
	}
	if len(in.ClientState) > businesscentral.MaxClientStateLength ||
		businesscentral.HasControl(in.ClientState) {
		return subscriptionBody{}, businesscentralerrors.ErrClientStateTooLong
	}
	return subscriptionBody{
		NotificationURL: target,
		Resource:        resource,
		ClientState:     in.ClientState,
	}, nil
}

func validSubscriptionID(id string) bool {
	if id == "" || len(id) > maxSubscriptionIDLength {
		return false
	}
	for idx := range len(id) {
		ch := id[idx]
		if !businesscentral.IsLetter(ch) && !businesscentral.IsDigit(ch) && ch != '-' {
			return false
		}
	}
	return true
}

func validEntityName(entity string) bool {
	if entity == "" || len(entity) > businesscentral.MaxDisplayNameLength {
		return false
	}
	for idx := range len(entity) {
		if !businesscentral.IsLetter(entity[idx]) {
			return false
		}
	}
	return true
}
