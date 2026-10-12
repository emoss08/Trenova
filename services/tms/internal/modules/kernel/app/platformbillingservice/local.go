package platformbillingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
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

var _ platformcatalogports.BillingProvider = (*LocalBillingProvider)(nil)

func NewLocalBillingProvider(p LocalBillingProviderParams) *LocalBillingProvider {
	return &LocalBillingProvider{
		catalog: p.Catalog,
	}
}

func (p *LocalBillingProvider) GetBillingSummary(
	_ context.Context,
	req *platformcatalogports.BillingSummaryRequest,
) (*platformcatalogports.BillingSummaryResult, error) {
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

	featureSummaries := make([]platformcatalogports.BillingFeatureSummary, 0, len(features))
	for i := range features {
		featureSummaries = append(featureSummaries, platformcatalogports.BillingFeatureSummary{
			FeatureKey: features[i].Key,
			Allowed:    true,
		})
	}

	usageSummaries := make([]platformcatalogports.BillingUsageSummary, 0, len(meters))
	for i := range meters {
		usageSummaries = append(usageSummaries, platformcatalogports.BillingUsageSummary{
			MeterKey: meters[i].Key,
			Unit:     meters[i].Unit,
		})
	}

	return &platformcatalogports.BillingSummaryResult{
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Active:         true,
		Reason:         communityReason,
		Plan: &platformcatalogports.BillingPlanSummary{
			ID:     communityPlanID,
			Key:    communityPlanID,
			Name:   communityName,
			Status: activeStatus,
		},
		Subscription: &platformcatalogports.BillingSubscriptionSummary{
			ID:     localID,
			PlanID: communityPlanID,
			Status: activeStatus,
		},
		Features:  featureSummaries,
		Usage:     usageSummaries,
		CheckedAt: checkedAt,
	}, nil
}
