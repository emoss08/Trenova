package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

func (s *Service) requireSamsara(ctx context.Context) error {
	return platformcatalogports.RequireFeature(
		ctx,
		s.features,
		platformcatalog.FeatureSamsaraIntegration,
	)
}

func (s *Service) samsaraOnPlan(ctx context.Context) (bool, error) {
	return s.features.Enabled(ctx, platformcatalog.FeatureSamsaraIntegration)
}
