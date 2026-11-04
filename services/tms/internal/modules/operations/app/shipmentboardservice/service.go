package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

	Repo            shipmentboardports.ShipmentBoardRepository
	QuickFilters    shipmentports.ShipmentQuickFilterBasisResolver
	Nova            llmports.TaskRunner
	Organizations   tenantports.OrganizationRepository
	Integrations    integrationports.IntegrationRepository
	IntegrationsSvc *integrationservice.Service
	Cache           shipmentboardports.ShipmentBoardCache
	Logger          *zap.Logger
}

type Service struct {
	repo          shipmentboardports.ShipmentBoardRepository
	quickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	nova          llmports.TaskRunner
	organizations tenantports.OrganizationRepository
	integrations  integrationports.IntegrationRepository
	maps          MapsReadiness
	cache         shipmentboardports.ShipmentBoardCache
	l             *zap.Logger
}

var (
	_ shipmentboardports.ShipmentBoardCapabilitiesReader = (*Service)(nil)
	_ shipmentboardports.ShipmentStageSummaryReader      = (*Service)(nil)
	_ shipmentboardports.ShipmentQuickFilterCounter      = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		repo:          p.Repo,
		quickFilters:  p.QuickFilters,
		nova:          p.Nova,
		organizations: p.Organizations,
		integrations:  p.Integrations,
		maps:          p.IntegrationsSvc,
		cache:         p.Cache,
		l:             p.Logger.Named("service.shipment-board"),
	}
}
