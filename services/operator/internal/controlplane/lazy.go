package controlplane

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type Lazy struct {
	doer *http.Client

	mu     sync.Mutex
	locate func() string
	api    *Client
	url    string
}

func NewLazy(doer *http.Client) *Lazy {
	return &Lazy{doer: doer}
}

func (c *Lazy) Locate(locate func() string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.locate = locate
}

func (c *Lazy) client() (*Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	url := ""
	if c.locate != nil {
		url = c.locate()
	}
	if url == "" {
		return nil, credential.ErrNotEnrolled
	}
	if c.api != nil && c.url == url {
		return c.api, nil
	}
	api, err := New(url, c.doer)
	if err != nil {
		return nil, err
	}
	c.api, c.url = api, url
	return api, nil
}

func (c *Lazy) Spec(ctx context.Context, etag string) (Desired, bool, error) {
	api, err := c.client()
	if err != nil {
		return Desired{}, false, err
	}
	return api.Spec(ctx, etag)
}

func (c *Lazy) ReportStatus(ctx context.Context, r *instanceapi.StatusReport) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.ReportStatus(ctx, r)
}

func (c *Lazy) ReportUsage(
	ctx context.Context,
	r instanceapi.UsageReport,
) (instanceapi.UsageAck, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.UsageAck{}, err
	}
	return api.ReportUsage(ctx, r)
}

func (c *Lazy) ReportTelemetry(ctx context.Context, r instanceapi.TelemetryReport) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.ReportTelemetry(ctx, r)
}

func (c *Lazy) Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.InstanceSecrets{}, err
	}
	return api.Secrets(ctx)
}

func (c *Lazy) Storage(ctx context.Context) (instanceapi.InstanceStorage, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.InstanceStorage{}, err
	}
	return api.Storage(ctx)
}

func (c *Lazy) Commands(ctx context.Context) (instanceapi.CommandList, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.CommandList{}, err
	}
	return api.Commands(ctx)
}

func (c *Lazy) CompleteCommand(
	ctx context.Context,
	id uuid.UUID,
	result instanceapi.CommandResult,
) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.CompleteCommand(ctx, id, result)
}

func (c *Lazy) EmailEvents(ctx context.Context) ([]instanceapi.EmailEvent, error) {
	api, err := c.client()
	if err != nil {
		return nil, err
	}
	return api.EmailEvents(ctx)
}

func (c *Lazy) AckEmailEvents(ctx context.Context, ids []uuid.UUID) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.AckEmailEvents(ctx, ids)
}

func (c *Lazy) RegisterLogBundle(
	ctx context.Context,
	reg *instanceapi.LogBundleRegistration,
) (uuid.UUID, error) {
	api, err := c.client()
	if err != nil {
		return uuid.Nil, err
	}
	return api.RegisterLogBundle(ctx, reg)
}

func (c *Lazy) UpdateLogBundle(
	ctx context.Context,
	id uuid.UUID,
	update instanceapi.LogBundleUpdate,
) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.UpdateLogBundle(ctx, id, update)
}

func (c *Lazy) Renew(ctx context.Context, csrPEM []byte) (credential.Issued, error) {
	api, err := c.client()
	if err != nil {
		return credential.Issued{}, err
	}
	return api.Renew(ctx, csrPEM)
}

func (c *Lazy) SharedTenants(ctx context.Context, etag string) (SharedDesired, bool, error) {
	api, err := c.client()
	if err != nil {
		return SharedDesired{}, false, err
	}
	return api.SharedTenants(ctx, etag)
}

func (c *Lazy) ReportSharedStatus(
	ctx context.Context,
	report *instanceapi.SharedStatusReport,
) (instanceapi.SharedReportAck, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.SharedReportAck{}, err
	}
	return api.ReportSharedStatus(ctx, report)
}

func (c *Lazy) ReportSharedUsage(
	ctx context.Context,
	report instanceapi.SharedUsageReport,
) (instanceapi.SharedUsageAck, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.SharedUsageAck{}, err
	}
	return api.ReportSharedUsage(ctx, report)
}
