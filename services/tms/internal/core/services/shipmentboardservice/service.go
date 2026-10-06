package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type MapsReadiness interface {
	GetClientRuntimeConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*integrationservice.RuntimeConfig, error)
}

type Params struct {
	fx.In

	Repo            repositories.ShipmentBoardRepository
	QuickFilters    services.ShipmentQuickFilterBasisResolver
	AIProviders     repositories.AIProviderRepository
	Organizations   repositories.OrganizationRepository
	Integrations    repositories.IntegrationRepository
	IntegrationsSvc *integrationservice.Service
}

type Service struct {
	repo          repositories.ShipmentBoardRepository
	quickFilters  services.ShipmentQuickFilterBasisResolver
	aiProviders   repositories.AIProviderRepository
	organizations repositories.OrganizationRepository
	integrations  repositories.IntegrationRepository
	maps          MapsReadiness
}

var (
	_ services.ShipmentBoardCapabilitiesReader = (*Service)(nil)
	_ services.ShipmentStageSummaryReader      = (*Service)(nil)
	_ services.ShipmentQuickFilterCounter      = (*Service)(nil)
	_ services.ShipmentFacetCounter            = (*Service)(nil)
)

type Dependencies struct {
	Repo          repositories.ShipmentBoardRepository
	QuickFilters  services.ShipmentQuickFilterBasisResolver
	AIProviders   repositories.AIProviderRepository
	Organizations repositories.OrganizationRepository
	Integrations  repositories.IntegrationRepository
	Maps          MapsReadiness
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Repo:          p.Repo,
		QuickFilters:  p.QuickFilters,
		AIProviders:   p.AIProviders,
		Organizations: p.Organizations,
		Integrations:  p.Integrations,
		Maps:          p.IntegrationsSvc,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		repo:          d.Repo,
		quickFilters:  d.QuickFilters,
		aiProviders:   d.AIProviders,
		organizations: d.Organizations,
		integrations:  d.Integrations,
		maps:          d.Maps,
	}
}
