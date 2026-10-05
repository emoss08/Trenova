package cloudquota

import (
	"github.com/emoss08/trenova/internal/cloud/cloudquota/quotacounterrepository"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Options(
	fx.Provide(quotacounterrepository.New),
	fx.Decorate(Decorate),
)

type DecorateParams struct {
	fx.In

	Default       services.QuotaGuard
	Config        *config.Config
	Plans         services.PlanService
	Counters      repositories.QuotaCounterRepository
	Subscriptions repositories.SubscriptionRepository
	DB            ports.DBConnection
	Logger        *zap.Logger
}

//nolint:gocritic // fx parameter objects are passed by value
func Decorate(p DecorateParams) services.QuotaGuard {
	if !p.Config.Platform.IsCloud() {
		return p.Default
	}

	return NewCloud(CloudConfig{
		Plans:         p.Plans,
		Counters:      p.Counters,
		Subscriptions: p.Subscriptions,
		DB:            p.DB,
		Logger:        p.Logger,
	})
}
