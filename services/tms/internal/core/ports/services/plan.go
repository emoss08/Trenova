package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PlanService interface {
	Resolve(ctx context.Context, orgID, buID pulid.ID) (*platformplan.ResolvedPlan, error)
	RequireCapability(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		capability platformplan.Capability,
	) error
	RequireWritable(ctx context.Context, tenantInfo pagination.TenantInfo) error
	EnforcesPlans() bool
	Invalidate(orgID pulid.ID)
}

type QuotaRequest struct {
	TenantInfo pagination.TenantInfo
	Meter      platformcatalog.MeterKey
	Quantity   int64
}

type QuotaDecision struct {
	Meter       platformcatalog.MeterKey `json:"meter"`
	Plan        platformplan.PlanKey     `json:"plan"`
	Window      platformplan.Window      `json:"window,omitempty"`
	Allowed     bool                     `json:"allowed"`
	Unlimited   bool                     `json:"unlimited"`
	Limit       int64                    `json:"limit"`
	Used        int64                    `json:"used"`
	Remaining   int64                    `json:"remaining"`
	Requested   int64                    `json:"requested"`
	WindowStart int64                    `json:"windowStart,omitempty"`
	WindowEnd   int64                    `json:"windowEnd,omitempty"`
}

type QuotaMeterUsage struct {
	Meter       platformcatalog.MeterKey `json:"meter"`
	Window      platformplan.Window      `json:"window"`
	Limit       int64                    `json:"limit"`
	Used        int64                    `json:"used"`
	Remaining   int64                    `json:"remaining"`
	WindowStart int64                    `json:"windowStart,omitempty"`
	WindowEnd   int64                    `json:"windowEnd,omitempty"`
}

type QuotaUsageSummary struct {
	OrganizationID         pulid.ID                  `json:"organizationId"`
	BusinessUnitID         pulid.ID                  `json:"businessUnitId"`
	Plan                   platformplan.PlanKey      `json:"plan"`
	PlanName               string                    `json:"planName"`
	Origin                 platformplan.Origin       `json:"origin"`
	Unlimited              bool                      `json:"unlimited"`
	SubscriptionID         pulid.ID                  `json:"subscriptionId,omitempty"`
	Status                 subscription.Status       `json:"status,omitempty"`
	TrialEndsAt            int64                     `json:"trialEndsAt,omitempty"`
	ReadOnlyUntil          int64                     `json:"readOnlyUntil,omitempty"`
	SubscribedAt           int64                     `json:"subscribedAt,omitempty"`
	RestrictedCapabilities []platformplan.Capability `json:"restrictedCapabilities"`
	Meters                 []QuotaMeterUsage         `json:"meters"`
	CheckedAt              int64                     `json:"checkedAt"`
}

type QuotaGuard interface {
	Enforce(ctx context.Context, req *QuotaRequest) error
	Check(ctx context.Context, req *QuotaRequest) (*QuotaDecision, error)
	Usage(ctx context.Context, tenantInfo pagination.TenantInfo) (*QuotaUsageSummary, error)
}
