package controlplane

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type Client interface {
	CheckFeature(
		context.Context,
		*services.FeatureCheckRequest,
	) (*services.FeatureCheckResult, error)
	CheckLimit(
		context.Context,
		*services.UsageLimitCheckRequest,
	) (*services.UsageLimitCheckResult, error)
	RecordUsage(context.Context, *services.UsageRecordRequest) (*services.UsageRecordResult, error)
}

type HTTPControlPlaneClientParams struct {
	fx.In

	Config     *config.Config
	HTTPClient *http.Client `optional:"true"`
}

type HTTPControlPlaneClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewHTTPControlPlaneClient(p HTTPControlPlaneClientParams) *HTTPControlPlaneClient {
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: p.Config.Platform.ControlPlane.GetTimeout(),
		}
	}

	return &HTTPControlPlaneClient{
		endpoint:   strings.TrimRight(p.Config.Platform.ControlPlane.Endpoint, "/"),
		apiKey:     p.Config.Platform.ControlPlane.APIKey,
		httpClient: httpClient,
	}
}

func (c *HTTPControlPlaneClient) CheckFeature(
	ctx context.Context,
	req *services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	var result services.FeatureCheckResult
	if err := c.post(ctx, "/v1/entitlements/check", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPControlPlaneClient) CheckLimit(
	ctx context.Context,
	req *services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	var result services.UsageLimitCheckResult
	if err := c.post(ctx, "/v1/usage/check-limit", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPControlPlaneClient) RecordUsage(
	ctx context.Context,
	req *services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	var result services.UsageRecordResult
	if err := c.post(ctx, "/v1/usage/record", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPControlPlaneClient) post(ctx context.Context, path string, payload, out any) error {
	body, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal control plane request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create control plane request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("control plane request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("control plane request failed with status %d", resp.StatusCode)
	}

	if err = sonic.ConfigDefault.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode control plane response: %w", err)
	}

	return nil
}

func failOpenAllowed(cfg *config.Config) bool {
	return cfg.App.IsDevelopment() && cfg.Platform.ControlPlane.FailOpenOnError
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func missingIdempotencyKeyError() error {
	return errortypes.NewValidationError(
		"idempotencyKey",
		errortypes.ErrInvalid,
		"idempotency key is required for cloud usage recording",
	)
}
