package platformbillingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

const (
	communityReason = "community_mode"
	communityPlanID = "community"
	communityName   = "Community"
	activeStatus    = "active"
	localID         = "local"
)

// LocalBillingProvider is the billing summary of a self-hosted install: one
// active community plan with nothing metered.
type LocalBillingProvider struct{}

var _ platformcatalogports.BillingProvider = (*LocalBillingProvider)(nil)

func NewLocalBillingProvider() *LocalBillingProvider {
	return &LocalBillingProvider{}
}

func (p *LocalBillingProvider) GetBillingSummary(
	_ context.Context,
	req *platformcatalogports.BillingSummaryRequest,
) (*platformcatalogports.BillingSummaryResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
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
		Features:  []platformcatalogports.BillingFeatureSummary{},
		Usage:     []platformcatalogports.BillingUsageSummary{},
		CheckedAt: checkedAt,
	}, nil
}
