package telematics

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/telematics/samsaraprovider"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FactoryParams struct {
	fx.In

	IntegrationRepo    repositories.IntegrationRepository
	IntegrationService *integrationservice.Service
	Logger             *zap.Logger
}

type Factory struct {
	integrationRepo    repositories.IntegrationRepository
	integrationService *integrationservice.Service
	l                  *zap.Logger
}

func NewFactory(p FactoryParams) services.TelematicsProviderFactory {
	return &Factory{
		integrationRepo:    p.IntegrationRepo,
		integrationService: p.IntegrationService,
		l:                  p.Logger.Named("telematics-provider-factory"),
	}
}

func (f *Factory) ProviderFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (services.TelematicsProvider, error) {
	integrations, err := f.integrationRepo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	for _, record := range integrations {
		if record.Category != integration.CategoryTelematics || !record.Enabled {
			continue
		}
		if !f.supported(record.Type) {
			continue
		}
		return f.ProviderOfType(ctx, tenantInfo, record.Type)
	}

	return nil, errortypes.NewBusinessError(
		"no telematics provider is enabled for this organization",
	)
}

func (f *Factory) ProviderOfType(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (services.TelematicsProvider, error) {
	switch typ { //nolint:exhaustive // only telematics-category providers supported; others hit default
	case integration.TypeSamsara:
		client, err := f.integrationService.SamsaraClient(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		return samsaraprovider.New(client), nil
	default:
		return nil, errortypes.NewBusinessError(
			string(typ) + " is not a supported telematics provider",
		)
	}
}

func (f *Factory) supported(typ integration.Type) bool {
	return typ == integration.TypeSamsara
}
