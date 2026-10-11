package telematics

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/telematics/samsaraprovider"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FactoryParams struct {
	fx.In

	IntegrationRepo    integrationports.IntegrationRepository
	IntegrationService *integrationservice.Service
	Logger             *zap.Logger
}

type Factory struct {
	integrationRepo    integrationports.IntegrationRepository
	integrationService *integrationservice.Service
	l                  *zap.Logger
}

func NewFactory(p FactoryParams) telematicsports.TelematicsProviderFactory {
	return &Factory{
		integrationRepo:    p.IntegrationRepo,
		integrationService: p.IntegrationService,
		l:                  p.Logger.Named("telematics-provider-factory"),
	}
}

func (f *Factory) ProviderFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (telematicsports.TelematicsProvider, error) {
	integrations, err := f.integrationRepo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	for _, record := range integrations {
		if record.Category != integration.CategoryTelematics || !record.Enabled {
			continue
		}
		if !record.Type.SupportsTelematics() {
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
) (telematicsports.TelematicsProvider, error) {
	switch typ { //nolint:exhaustive // only telematics-category providers supported; others hit default
	case integration.TypeSamsara:
		client, err := f.integrationService.SamsaraClient(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		return samsaraprovider.New(client), nil
	default:
		return nil, errortypes.NewBusinessError(
			"{0} is not a supported telematics provider", string(typ),
		)
	}
}
