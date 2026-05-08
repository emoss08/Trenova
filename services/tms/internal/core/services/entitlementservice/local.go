package entitlementservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type LocalEntitlementProviderParams struct {
	fx.In

	Registry *platformcatalog.Registry
}

type LocalEntitlementProvider struct {
	registry *platformcatalog.Registry
}

func NewLocalEntitlementProvider(p LocalEntitlementProviderParams) *LocalEntitlementProvider {
	return &LocalEntitlementProvider{
		registry: p.Registry,
	}
}

func (p *LocalEntitlementProvider) CheckFeature(
	_ context.Context,
	req *services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	if _, ok := p.registry.GetFeature(req.FeatureKey); !ok {
		return &services.FeatureCheckResult{
			FeatureKey: req.FeatureKey,
			Allowed:    false,
			Reason:     "feature_not_found",
			CheckedAt:  checkedAt,
		}, nil
	}

	return &services.FeatureCheckResult{
		FeatureKey: req.FeatureKey,
		Allowed:    true,
		Reason:     "community_mode",
		CheckedAt:  checkedAt,
	}, nil
}

func (p *LocalEntitlementProvider) ListEntitlements(
	_ context.Context,
	req *services.EntitlementsRequest,
) (*services.EntitlementsResult, error) {
	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	features := p.registry.ListFeatures()
	results := make([]services.FeatureCheckResult, 0, len(features))
	for i := range features {
		feature := features[i]
		results = append(results, services.FeatureCheckResult{
			FeatureKey: feature.Key,
			Allowed:    true,
			Reason:     "community_mode",
			CheckedAt:  checkedAt,
		})
	}

	return &services.EntitlementsResult{
		Features:  results,
		CheckedAt: checkedAt,
	}, nil
}
