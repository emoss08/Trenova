package usageservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

type NoopUsageProvider struct{}

func NewNoopUsageProvider() *NoopUsageProvider {
	return &NoopUsageProvider{}
}

func (p *NoopUsageProvider) CheckLimit(
	_ context.Context,
	req *services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	return &services.UsageLimitCheckResult{
		MeterKey:  req.MeterKey,
		Allowed:   true,
		Reason:    "noop_usage_provider",
		CheckedAt: checkedAt,
	}, nil
}

func (p *NoopUsageProvider) RecordUsage(
	_ context.Context,
	req *services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	recordedAt := req.RecordedAt
	if recordedAt == 0 {
		recordedAt = time.Now().Unix()
	}

	return &services.UsageRecordResult{
		MeterKey:       req.MeterKey,
		Recorded:       false,
		Quantity:       req.Quantity,
		RecordedAt:     recordedAt,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}
