package quotaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type UnlimitedQuotaGuard struct{}

var _ services.QuotaGuard = (*UnlimitedQuotaGuard)(nil)

func NewUnlimited() *UnlimitedQuotaGuard {
	return &UnlimitedQuotaGuard{}
}

func (g *UnlimitedQuotaGuard) Enforce(_ context.Context, req *services.QuotaRequest) error {
	if req == nil {
		return ErrRequestRequired
	}
	if req.Quantity < 0 {
		return ErrInvalidQuantity
	}

	return nil
}

func (g *UnlimitedQuotaGuard) Check(
	_ context.Context,
	req *services.QuotaRequest,
) (*services.QuotaDecision, error) {
	if req == nil {
		return nil, ErrRequestRequired
	}

	return &services.QuotaDecision{
		Meter:     req.Meter,
		Plan:      platformplan.PlanKeyUnlimited,
		Allowed:   true,
		Unlimited: true,
		Requested: req.Quantity,
	}, nil
}

func (g *UnlimitedQuotaGuard) Usage(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.QuotaUsageSummary, error) {
	plan := platformplan.Unlimited()

	return &services.QuotaUsageSummary{
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Plan:                   plan.Key,
		PlanName:               plan.Name,
		Origin:                 platformplan.OriginSelfHosted,
		Unlimited:              true,
		RestrictedCapabilities: []platformplan.Capability{},
		Meters:                 []services.QuotaMeterUsage{},
		CheckedAt:              time.Now().Unix(),
	}, nil
}
