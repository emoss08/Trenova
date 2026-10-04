package usageservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

const (
	ReasonQuotaExceeded  = "quota_exceeded"
	ReasonWithinPlan     = "within_plan_limit"
	ReasonUnlimitedPlan  = "unlimited_plan"
	ReasonNotPlanMetered = "not_plan_metered"
	ReasonDerivedUsage   = "derived_from_records"
)

var planMeteredUsage = map[platformcatalog.MeterKey]struct{}{
	platformcatalog.MeterDocumentUploads:      {},
	platformcatalog.MeterDocumentStorageBytes: {},
	platformcatalog.MeterDocumentFileBytes:    {},
}

type LocalPlanUsageProviderParams struct {
	fx.In

	Quota services.QuotaGuard
}

type LocalPlanUsageProvider struct {
	quota services.QuotaGuard
}

var _ services.UsageProvider = (*LocalPlanUsageProvider)(nil)

func NewLocalPlanUsageProvider(p LocalPlanUsageProviderParams) *LocalPlanUsageProvider {
	return &LocalPlanUsageProvider{quota: p.Quota}
}

func (p *LocalPlanUsageProvider) CheckLimit(
	ctx context.Context,
	req *services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	if _, metered := planMeteredUsage[req.MeterKey]; !metered {
		return &services.UsageLimitCheckResult{
			MeterKey:  req.MeterKey,
			Allowed:   true,
			Reason:    ReasonNotPlanMetered,
			CheckedAt: checkedAt,
		}, nil
	}

	decision, err := p.quota.Check(ctx, &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.OrganizationID,
			BuID:   req.BusinessUnitID,
			UserID: req.UserID,
		},
		Meter:    req.MeterKey,
		Quantity: max(req.Quantity, 0),
	})
	if err != nil {
		return nil, err
	}

	result := &services.UsageLimitCheckResult{
		MeterKey:  req.MeterKey,
		Allowed:   decision.Allowed,
		Limit:     decision.Limit,
		Used:      decision.Used,
		Remaining: decision.Remaining,
		Plan:      decision.Plan.String(),
		CheckedAt: checkedAt,
	}

	switch {
	case !decision.Allowed:
		result.Reason = ReasonQuotaExceeded
	case decision.Unlimited:
		result.Reason = ReasonUnlimitedPlan
	default:
		result.Reason = ReasonWithinPlan
	}

	return result, nil
}

func (p *LocalPlanUsageProvider) RecordUsage(
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
