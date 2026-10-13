package businesscentral

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	subscriptionsEntity     = "subscriptions"
	subscriptionResourceAPI = "/" + apiSegment + "/"
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
	SubscriptionID       string   `json:"subscriptionId"`
	NotificationURL      string   `json:"notificationUrl"`
	Resource             string   `json:"resource"`
	Timestamp            int64    `json:"timestamp"`
	UserID               string   `json:"userId"`
	ClientState          string   `json:"clientState"`
	ExpirationDateTime   wireTime `json:"expirationDateTime"`
	SystemCreatedAt      wireTime `json:"systemCreatedAt"`
	LastModifiedDateTime wireTime `json:"lastModifiedDateTime"`
	ETag                 string   `json:"@odata.etag"`
}

func (w *wireSubscription) subscription() Subscription {
	return Subscription{
		ID:                 w.SubscriptionID,
		NotificationURL:    w.NotificationURL,
		Resource:           w.Resource,
		Timestamp:          w.Timestamp,
		UserID:             emptyGUID(w.UserID),
		ClientState:        w.ClientState,
		ExpirationDateTime: w.ExpirationDateTime.time(),
		CreatedAt:          w.SystemCreatedAt.time(),
		LastModified:       w.LastModifiedDateTime.time(),
		ETag:               w.ETag,
	}
}

type subscriptionBody struct {
	NotificationURL string `json:"notificationUrl"`
	Resource        string `json:"resource"`
	ClientState     string `json:"clientState,omitempty"`
}

func SubscriptionResource(ref CompanyRef, entity string) string {
	normalized, err := NewCompanyRef(ref.TenantID, ref.Environment, ref.CompanyID)
	if err != nil || !validEntityName(entity) {
		return ""
	}
	return subscriptionResourceAPI + companiesEntity + "(" + normalized.CompanyID + ")/" + entity
}

func (c *Client) Subscriptions(ctx context.Context) ([]Subscription, error) {
	return collect(ctx, c.core, &listCall{
		endpoint: "subscriptions",
		path:     c.envPath + "/" + subscriptionsEntity,
		unpaged:  true,
	}, (*wireSubscription).subscription)
}

func (c *Client) CreateSubscription(
	ctx context.Context,
	in *SubscriptionInput,
) (*Subscription, error) {
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: "subscriptions-create",
		method:   http.MethodPost,
		path:     c.envPath + "/" + subscriptionsEntity,
		body:     body,
		expected: []int{http.StatusCreated},
	}, (*wireSubscription).subscription)
}

func (c *Client) RenewSubscription(
	ctx context.Context,
	subscriptionID, etag string,
	in *SubscriptionInput,
) (*Subscription, error) {
	path, err := c.subscriptionPath(subscriptionID)
	if err != nil {
		return nil, err
	}
	match, err := ifMatch(etag)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: "subscriptions-renew",
		method:   http.MethodPatch,
		path:     path,
		ifMatch:  match,
		body:     body,
	}, (*wireSubscription).subscription)
}

func (c *Client) DeleteSubscription(ctx context.Context, subscriptionID, etag string) error {
	path, err := c.subscriptionPath(subscriptionID)
	if err != nil {
		return err
	}
	return c.remove(ctx, "subscriptions-delete", path, etag)
}

func (c *Client) subscriptionPath(subscriptionID string) (string, error) {
	id := strings.TrimSpace(subscriptionID)
	if !validSubscriptionID(id) {
		return "", ErrInvalidSubscriptionID
	}
	return c.envPath + "/" + subscriptionsEntity + "('" + id + "')", nil
}

func (in *SubscriptionInput) body() (subscriptionBody, error) {
	target := strings.TrimSpace(in.NotificationURL)
	parsed, err := url.Parse(target)
	if err != nil || len(target) > maxNotificationURLSize || parsed.Scheme != httpsScheme ||
		parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return subscriptionBody{}, ErrNotificationURL
	}
	resource := strings.TrimSpace(in.Resource)
	if resource == "" {
		return subscriptionBody{}, ErrResourceRequired
	}
	if !strings.HasPrefix(resource, subscriptionResourceAPI) ||
		len(resource) > maxResourceLength || hasControl(resource) ||
		strings.ContainsAny(resource, " ?#") {
		return subscriptionBody{}, ErrResourceRequired
	}
	if len(in.ClientState) > MaxClientStateLength || hasControl(in.ClientState) {
		return subscriptionBody{}, ErrClientStateTooLong
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
		if !isLetter(ch) && !isDigit(ch) && ch != '-' {
			return false
		}
	}
	return true
}

func validEntityName(entity string) bool {
	if entity == "" || len(entity) > maxDisplayNameLength {
		return false
	}
	for idx := range len(entity) {
		if !isLetter(entity[idx]) {
			return false
		}
	}
	return true
}
