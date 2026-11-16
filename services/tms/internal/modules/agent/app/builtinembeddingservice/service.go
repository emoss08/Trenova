package builtinembeddingservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

var _ aiproviderports.BuiltInEmbedding = (*Service)(nil)

type Params struct {
	fx.In

	Managed  *managedconfig.Config
	Features platformcatalogports.FeatureGate
}

type Service struct {
	service    aiprovider.BuiltInEmbeddingService
	configured bool
	features   platformcatalogports.FeatureGate
}

func New(p Params) *Service {
	embedding, configured := p.Managed.BuiltInEmbedding()

	return &Service{
		service: aiprovider.BuiltInEmbeddingService{
			URL:   embedding.URL,
			Model: embedding.Model,
		},
		configured: configured,
		features:   p.Features,
	}
}

func AsPort(s *Service) aiproviderports.BuiltInEmbedding { return s }

func (s *Service) Provider(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*aiprovider.Provider, bool, error) {
	if !s.configured {
		return nil, false, nil
	}

	included, err := s.features.Enabled(ctx, platformcatalog.FeatureAIDesk)
	if err != nil {
		return nil, false, fmt.Errorf("read whether the plan includes Trenova AI: %w", err)
	}
	if !included {
		return nil, false, nil
	}

	return aiprovider.BuiltInEmbedding(tenant.OrgID, tenant.BuID, s.service), true, nil
}
