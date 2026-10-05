package controlplane

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

type CloudUsageProviderParams struct {
	fx.In

	Config *config.Config
	Client Client
}

type CloudUsageProvider struct {
	cfg    *config.Config
	client Client
}

func NewCloudUsageProvider(p CloudUsageProviderParams) *CloudUsageProvider {
	return &CloudUsageProvider{
		cfg:    p.Config,
		client: p.Client,
	}
}

func (p *CloudUsageProvider) CheckLimit(
	ctx context.Context,
	req *services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	result, err := p.client.CheckLimit(ctx, req)
	if err == nil {
		return result, nil
	}
	if !failOpenAllowed(p.cfg) {
		return nil, err
	}

	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = nowUnix()
	}

	return &services.UsageLimitCheckResult{
		MeterKey:  req.MeterKey,
		Allowed:   true,
		Reason:    fmt.Sprintf("fail_open:%s", err.Error()),
		CheckedAt: checkedAt,
		FailOpen:  true,
	}, nil
}

func (p *CloudUsageProvider) RecordUsage(
	ctx context.Context,
	req *services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	if req.IdempotencyKey == "" {
		return nil, missingIdempotencyKeyError()
	}

	result, err := p.client.RecordUsage(ctx, req)
	if err == nil {
		return result, nil
	}
	if !failOpenAllowed(p.cfg) {
		return nil, err
	}

	recordedAt := req.RecordedAt
	if recordedAt == 0 {
		recordedAt = nowUnix()
	}

	return &services.UsageRecordResult{
		MeterKey:       req.MeterKey,
		Recorded:       false,
		Quantity:       req.Quantity,
		RecordedAt:     recordedAt,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}
