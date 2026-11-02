package entitlementservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

const reasonCommunityMode = "community_mode"

// LocalEntitlementProvider is the entitlements of a self-hosted install: every
// feature is allowed.
type LocalEntitlementProvider struct{}

var _ platformcatalogports.EntitlementProvider = (*LocalEntitlementProvider)(nil)

func NewLocalEntitlementProvider() *LocalEntitlementProvider {
	return &LocalEntitlementProvider{}
}

func (p *LocalEntitlementProvider) CheckFeature(
	_ context.Context,
	req *platformcatalogports.FeatureCheckRequest,
) (*platformcatalogports.FeatureCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	return &platformcatalogports.FeatureCheckResult{
		FeatureKey: req.FeatureKey,
		Allowed:    true,
		Reason:     reasonCommunityMode,
		CheckedAt:  checkedAt,
	}, nil
}

func (p *LocalEntitlementProvider) ListEntitlements(
	_ context.Context,
	req *platformcatalogports.EntitlementsRequest,
) (*platformcatalogports.EntitlementsResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	return &platformcatalogports.EntitlementsResult{
		Features:  []platformcatalogports.FeatureCheckResult{},
		CheckedAt: checkedAt,
	}, nil
}
