package platformbillingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

const (
	communityReason = "community_mode"
	communityPlanID = "community"
	communityName   = "Community"
	activeStatus    = "active"
	localID         = "local"
)

type LocalBillingProviderParams struct {
	fx.In

	Catalog platformcatalog.FeatureCatalog `optional:"true"`
}

type LocalBillingProvider struct {
	catalog platformcatalog.FeatureCatalog
}

var _ services.BillingProvider = (*LocalBillingProvider)(nil)

func NewLocalBillingProvider(p LocalBillingProviderParams) *LocalBillingProvider {
	return &LocalBillingProvider{
		catalog: p.Catalog,
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

	var (
		features []platformcatalog.Feature
		meters   []platformcatalog.Meter
	)
	if p.catalog != nil {
		features = p.catalog.ListFeatures()
		meters = p.catalog.ListMeters()
	}

	featureSummaries := make([]services.BillingFeatureSummary, 0, len(features))
	for i := range features {
		featureSummaries = append(featureSummaries, services.BillingFeatureSummary{
			FeatureKey: features[i].Key,
			Allowed:    true,
		})
	}

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
		Reason:         communityReason,
		Plan: &services.BillingPlanSummary{
			ID:     communityPlanID,
			Key:    communityPlanID,
			Name:   communityName,
			Status: activeStatus,
		},
		Subscription: &services.BillingSubscriptionSummary{
			ID:     localID,
			PlanID: communityPlanID,
			Status: activeStatus,
		},
		Features:  featureSummaries,
		Usage:     usageSummaries,
		CheckedAt: checkedAt,
	}, nil
}
