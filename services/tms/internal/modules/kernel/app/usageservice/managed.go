package usageservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

const PlanManaged = "managed"

type ManagedUsageProvider struct {
	NoopUsageProvider
	gate managedusageports.Gate
}

var _ platformcatalogports.UsageProvider = (*ManagedUsageProvider)(nil)

func NewManagedUsageProvider(gate managedusageports.Gate) *ManagedUsageProvider {
	return &ManagedUsageProvider{gate: gate}
}

func (p *ManagedUsageProvider) CheckLimit(
	ctx context.Context,
	req *platformcatalogports.UsageLimitCheckRequest,
) (*platformcatalogports.UsageLimitCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}
	d, err := p.gate.Decide(ctx, req.MeterKey, req.Quantity)
	if err != nil {
		return nil, err
	}
	return &platformcatalogports.UsageLimitCheckResult{
		MeterKey:  req.MeterKey,
		Allowed:   d.Allowed,
		Reason:    managedReason(&d),
		Limit:     d.Limit,
		Used:      d.Used,
		Remaining: d.Remaining(),
		Plan:      PlanManaged,
		CheckedAt: checkedAt,
	}, nil
}

func managedReason(d *managedusage.Decision) string {
	switch {
	case !d.Allowed:
		return ReasonQuotaExceeded
	case d.Managed:
		return ReasonWithinPlan
	default:
		return ReasonNotPlanMetered
	}
}
