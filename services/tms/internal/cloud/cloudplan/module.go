package cloudplan

import (
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/cloudplan/subscriptionrepository"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Options(
	fx.Provide(NewCatalog, subscriptionrepository.New),
	fx.Decorate(Decorate),
)

type CatalogParams struct {
	fx.In

	Config *config.Config
}

func NewCatalog(p CatalogParams) (*platformplan.Catalog, error) {
	return platformplan.NewCatalog(platformplan.ParseLimitOverrides(
		cloudconfig.From(p.Config).Cloud.FreePlan.GetLimitOverrides(),
	))
}

type DecorateParams struct {
	fx.In

	Default       services.PlanService
	Config        *config.Config
	Catalog       *platformplan.Catalog
	Subscriptions repositories.SubscriptionRepository
	Logger        *zap.Logger
}

func Decorate(p DecorateParams) services.PlanService {
	if !p.Config.Platform.IsCloud() {
		return p.Default
	}

	return NewCloud(CloudConfig{
		Catalog:       p.Catalog,
		Subscriptions: p.Subscriptions,
		Logger:        p.Logger,
	})
}
