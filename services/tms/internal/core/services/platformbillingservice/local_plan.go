package platformbillingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
)

const (
	reasonCloudPlan     = "cloud_plan"
	reasonCloudInternal = "cloud_internal"
	internalPlanID      = "internal"
	activeStatus        = "active"
)

type LocalPlanBillingProviderParams struct {
	fx.In

	Registry *platformcatalog.Registry
	Quota    services.QuotaGuard
}

type LocalPlanBillingProvider struct {
	registry *platformcatalog.Registry
	quota    services.QuotaGuard
}

var _ services.BillingProvider = (*LocalPlanBillingProvider)(nil)

func NewLocalPlanBillingProvider(p LocalPlanBillingProviderParams) *LocalPlanBillingProvider {
	return &LocalPlanBillingProvider{
		registry: p.Registry,
		quota:    p.Quota,
	}
}

func (p *LocalPlanBillingProvider) GetBillingSummary(
	ctx context.Context,
	req *services.BillingSummaryRequest,
) (*services.BillingSummaryResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	usage, err := p.quota.Usage(ctx, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.UserID,
	})
	if err != nil {
		return nil, err
	}

	managed := usage.Origin == platformplan.OriginSubscription
	status := activeStatus
	if managed && usage.Status != "" {
		status = usage.Status.String()
	}

	result := &services.BillingSummaryResult{
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Active:         !managed || usage.Status.AllowsWrites(),
		Reason:         reasonCloudInternal,
		Plan: &services.BillingPlanSummary{
			ID:     usage.Plan.String(),
			Key:    usage.Plan.String(),
			Name:   usage.PlanName,
			Status: status,
		},
		Subscription: &services.BillingSubscriptionSummary{
			ID:     internalPlanID,
			PlanID: usage.Plan.String(),
			Status: status,
		},
		Features:     p.features(),
		Usage:        p.usage(usage),
		Restrictions: sliceutils.Strings(usage.RestrictedCapabilities),
		CheckedAt:    checkedAt,
	}

	if managed {
		result.Reason = reasonCloudPlan
		result.Subscription.ID = usage.SubscriptionID.String()
		result.Subscription.CurrentPeriodStart = usage.SubscribedAt
		result.Subscription.CurrentPeriodEnd = usage.TrialEndsAt
		result.Subscription.TrialEndsAt = usage.TrialEndsAt
		result.Subscription.ReadOnlyUntil = usage.ReadOnlyUntil
	}

	return result, nil
}

func (p *LocalPlanBillingProvider) features() []services.BillingFeatureSummary {
	features := p.registry.ListFeatures()
	summaries := make([]services.BillingFeatureSummary, 0, len(features))
	for i := range features {
		summaries = append(summaries, services.BillingFeatureSummary{
			FeatureKey: features[i].Key,
			Allowed:    true,
		})
	}

	return summaries
}

func (p *LocalPlanBillingProvider) usage(
	usage *services.QuotaUsageSummary,
) []services.BillingUsageSummary {
	meters := p.registry.ListMeters()
	units := make(map[platformcatalog.MeterKey]string, len(meters))
	for i := range meters {
		units[meters[i].Key] = meters[i].Unit
	}

	if usage.Unlimited || len(usage.Meters) == 0 {
		summaries := make([]services.BillingUsageSummary, 0, len(meters))
		for i := range meters {
			summaries = append(summaries, services.BillingUsageSummary{
				MeterKey: meters[i].Key,
				Unit:     meters[i].Unit,
			})
		}
		return summaries
	}

	summaries := make([]services.BillingUsageSummary, 0, len(usage.Meters))
	for _, meter := range usage.Meters {
		summaries = append(summaries, services.BillingUsageSummary{
			MeterKey:    meter.Meter,
			Unit:        units[meter.Meter],
			Limit:       meter.Limit,
			Used:        meter.Used,
			Remaining:   meter.Remaining,
			WindowStart: meter.WindowStart,
			WindowEnd:   meter.WindowEnd,
			Window:      meter.Window.String(),
		})
	}

	return summaries
}
