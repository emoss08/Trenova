package integrationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
)

func planFeature(typ integration.Type) (platformcatalog.FeatureKey, bool) {
	switch typ {
	case integration.TypeSamsara:
		return platformcatalog.FeatureSamsaraIntegration, true
	case integration.TypeGoogleMaps:
		return platformcatalog.FeatureGoogleMapsIntegration, true
	default:
		return "", false
	}
}

func (s *Service) requireIntegration(ctx context.Context, typ integration.Type) error {
	if err := platformcatalogports.RequireFeature(
		ctx,
		s.features,
		platformcatalog.FeatureAdministration,
	); err != nil {
		return err
	}
	feature, ok := planFeature(typ)
	if !ok {
		return nil
	}
	return platformcatalogports.RequireFeature(ctx, s.features, feature)
}

func (s *Service) integrationOnPlan(ctx context.Context, typ integration.Type) (bool, error) {
	allowed, err := s.features.Enabled(ctx, platformcatalog.FeatureAdministration)
	if err != nil || !allowed {
		return false, err
	}
	feature, ok := planFeature(typ)
	if !ok {
		return true, nil
	}
	return s.features.Enabled(ctx, feature)
}
