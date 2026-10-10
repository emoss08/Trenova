package quotaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type UnlimitedQuotaGuard struct{}

var _ platformplanports.QuotaGuard = (*UnlimitedQuotaGuard)(nil)

func NewUnlimited() *UnlimitedQuotaGuard {
	return &UnlimitedQuotaGuard{}
}

func (g *UnlimitedQuotaGuard) Enforce(_ context.Context, req *platformplanports.QuotaRequest) error {
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
	req *platformplanports.QuotaRequest,
) (*platformplanports.QuotaDecision, error) {
	if req == nil {
		return nil, ErrRequestRequired
	}

	return &platformplanports.QuotaDecision{
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
) (*platformplanports.QuotaUsageSummary, error) {
	plan := platformplan.Unlimited()

	return &platformplanports.QuotaUsageSummary{
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Plan:                   plan.Key,
		PlanName:               plan.Name,
		Origin:                 platformplan.OriginSelfHosted,
		Unlimited:              true,
		RestrictedCapabilities: []platformplan.Capability{},
		Meters:                 []platformplanports.QuotaMeterUsage{},
		CheckedAt:              time.Now().Unix(),
	}, nil
}
