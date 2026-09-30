package xero

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

type Connection struct {
	ID          string
	AuthEventID string
	TenantID    string
	TenantType  string
	TenantName  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ConnectionsClient struct {
	transport *restx.Client
}

type wireConnection struct {
	ID             string   `json:"id"`
	AuthEventID    string   `json:"authEventId"`
	TenantID       string   `json:"tenantId"`
	TenantType     string   `json:"tenantType"`
	TenantName     string   `json:"tenantName"`
	CreatedDateUTC wireTime `json:"createdDateUtc"`
	UpdatedDateUTC wireTime `json:"updatedDateUtc"`
}

func NewConnectionsClient(accessToken string, opts ...Option) (*ConnectionsClient, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, ErrAccessTokenRequired
	}

	settings := resolveOptions(opts)
	transport, err := restx.New(restx.Config{
		BaseURL:      settings.baseURL,
		Timeout:      settings.timeout,
		UserAgent:    settings.userAgent,
		HTTPClient:   settings.httpClient,
		Headers:      map[string]string{authorizationHeader: "Bearer " + token},
		Retry:        settings.retry,
		Observer:     settings.observer,
		ErrorDecoder: decodeAPIError,
	})
	if err != nil {
		return nil, fmt.Errorf("xero: configure connections transport: %w", err)
	}
	return &ConnectionsClient{transport: transport}, nil
}

func (c *ConnectionsClient) List(ctx context.Context, authEventID string) ([]Connection, error) {
	var query url.Values
	if eventID := strings.TrimSpace(authEventID); eventID != "" {
		query = url.Values{"authEventId": {eventID}}
	}

	var out []wireConnection
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "connections",
		Method:   http.MethodGet,
		Path:     connectionsPath,
		Query:    query,
		Out:      &out,
	}); err != nil {
		redactAPIError(err, c.transport.Redact)
		return nil, err
	}

	connections := make([]Connection, 0, len(out))
	for idx := range out {
		wire := &out[idx]
		connections = append(connections, Connection{
			ID:          wire.ID,
			AuthEventID: wire.AuthEventID,
			TenantID:    wire.TenantID,
			TenantType:  wire.TenantType,
			TenantName:  wire.TenantName,
			CreatedAt:   wire.CreatedDateUTC.time(),
			UpdatedAt:   wire.UpdatedDateUTC.time(),
		})
	}
	return connections, nil
}

func (c *ConnectionsClient) Delete(ctx context.Context, connectionID string) error {
	id, err := guid(connectionID)
	if err != nil {
		return err
	}
	_, err = c.transport.Do(ctx, &restx.Request{
		Endpoint:       "connections-delete",
		Method:         http.MethodDelete,
		Path:           connectionsPath + "/" + url.PathEscape(id),
		ExpectedStatus: []int{http.StatusNoContent, http.StatusOK},
	})
	if err != nil {
		redactAPIError(err, c.transport.Redact)
	}
	return err
}
