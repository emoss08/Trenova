package platformbillingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type LocalBillingProviderParams struct {
	fx.In

	Registry *platformcatalog.Registry
}

type LocalBillingProvider struct {
	registry *platformcatalog.Registry
}

func NewLocalBillingProvider(p LocalBillingProviderParams) *LocalBillingProvider {
	return &LocalBillingProvider{
		registry: p.Registry,
	}
}

func (p *LocalBillingProvider) GetBillingSummary(
	_ context.Context,
	req *services.BillingSummaryRequest,
) (*services.BillingSummaryResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	features := p.registry.ListFeatures()
	featureSummaries := make([]services.BillingFeatureSummary, 0, len(features))
	for i := range features {
		featureSummaries = append(featureSummaries, services.BillingFeatureSummary{
			FeatureKey: features[i].Key,
			Allowed:    true,
		})
	}

	meters := p.registry.ListMeters()
	usageSummaries := make([]services.BillingUsageSummary, 0, len(meters))
	for i := range meters {
		usageSummaries = append(usageSummaries, services.BillingUsageSummary{
			MeterKey: meters[i].Key,
			Unit:     meters[i].Unit,
		})
	}

	return &services.BillingSummaryResult{
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Active:         true,
		Reason:         "community_mode",
		Plan: &services.BillingPlanSummary{
			ID:     "community",
			Key:    "community",
			Name:   "Community",
			Status: "active",
		},
		Subscription: &services.BillingSubscriptionSummary{
			ID:     "local",
			PlanID: "community",
			Status: "active",
		},
		Features:  featureSummaries,
		Usage:     usageSummaries,
		CheckedAt: checkedAt,
	}, nil
}
