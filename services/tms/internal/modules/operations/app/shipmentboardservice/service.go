package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
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

	Repo            shipmentports.ShipmentBoardRepository
	QuickFilters    shipmentports.ShipmentQuickFilterBasisResolver
	AIProviders     aiproviderports.AIProviderRepository
	Organizations   tenantports.OrganizationRepository
	Integrations    integrationports.IntegrationRepository
	IntegrationsSvc *integrationservice.Service
}

type Service struct {
	repo          shipmentports.ShipmentBoardRepository
	quickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	aiProviders   aiproviderports.AIProviderRepository
	organizations tenantports.OrganizationRepository
	integrations  integrationports.IntegrationRepository
	maps          MapsReadiness
}

var (
	_ shipmentports.ShipmentBoardCapabilitiesReader = (*Service)(nil)
	_ shipmentports.ShipmentStageSummaryReader      = (*Service)(nil)
	_ shipmentports.ShipmentQuickFilterCounter      = (*Service)(nil)
)

type Dependencies struct {
	Repo          shipmentports.ShipmentBoardRepository
	QuickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	AIProviders   aiproviderports.AIProviderRepository
	Organizations tenantports.OrganizationRepository
	Integrations  integrationports.IntegrationRepository
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
