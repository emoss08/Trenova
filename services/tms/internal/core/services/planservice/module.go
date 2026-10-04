package planservice

import (
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("plan-service", fx.Provide(NewCatalog, New))

type CatalogParams struct {
	fx.In

	Config *config.Config
}

func NewCatalog(p CatalogParams) (*platformplan.Catalog, error) {
	return platformplan.NewCatalog(platformplan.ParseLimitOverrides(
		p.Config.Platform.Cloud.FreePlan.GetLimitOverrides(),
	))
}

type Params struct {
	fx.In

	Config        *config.Config
	Catalog       *platformplan.Catalog
	Subscriptions repositories.SubscriptionRepository
	Logger        *zap.Logger
}

func New(p Params) services.PlanService {
	if !p.Config.Platform.IsCloud() {
		return NewUnlimited()
	}

	return NewCloud(CloudConfig{
		Catalog:       p.Catalog,
		Subscriptions: p.Subscriptions,
		Logger:        p.Logger,
	})
}
