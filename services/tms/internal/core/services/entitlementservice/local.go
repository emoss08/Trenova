package entitlementservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"go.uber.org/fx"
)

const (
	reasonCommunityMode   = "community_mode"
	reasonFeatureNotFound = "feature_not_found"
)

type LocalEntitlementProviderParams struct {
	fx.In

	Catalog platformcatalog.FeatureCatalog `optional:"true"`
}

type LocalEntitlementProvider struct {
	catalog platformcatalog.FeatureCatalog
}

var _ platformcatalogports.EntitlementProvider = (*LocalEntitlementProvider)(nil)

func NewLocalEntitlementProvider(p LocalEntitlementProviderParams) *LocalEntitlementProvider {
	return &LocalEntitlementProvider{
		catalog: p.Catalog,
	}
}

func (p *LocalEntitlementProvider) CheckFeature(
	_ context.Context,
	req *platformcatalogports.FeatureCheckRequest,
) (*platformcatalogports.FeatureCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	if p.catalog != nil {
		if _, ok := p.catalog.GetFeature(req.FeatureKey); !ok {
			return &platformcatalogports.FeatureCheckResult{
				FeatureKey: req.FeatureKey,
				Allowed:    false,
				Reason:     reasonFeatureNotFound,
				CheckedAt:  checkedAt,
			}, nil
		}
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

	var features []platformcatalog.Feature
	if p.catalog != nil {
		features = p.catalog.ListFeatures()
	}

	results := make([]platformcatalogports.FeatureCheckResult, 0, len(features))
	for i := range features {
		results = append(results, platformcatalogports.FeatureCheckResult{
			FeatureKey: features[i].Key,
			Allowed:    true,
			Reason:     reasonCommunityMode,
			CheckedAt:  checkedAt,
		})
	}

	return &platformcatalogports.EntitlementsResult{
		Features:  results,
		CheckedAt: checkedAt,
	}, nil
}
