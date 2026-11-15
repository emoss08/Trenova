package edisetup

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

func (s *Service) requireEDIIntegration(ctx context.Context) error {
	return platformcatalogports.RequireFeature(
		ctx,
		s.features,
		platformcatalog.FeatureEDIIntegration,
	)
}
