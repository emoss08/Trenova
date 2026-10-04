package quotaservice

import (
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("quota-service", fx.Provide(New))

type Params struct {
	fx.In

	Config   *config.Config
	Plans    services.PlanService
	Counters repositories.QuotaCounterRepository
	DB       ports.DBConnection
	Logger   *zap.Logger
}

func New(p Params) services.QuotaGuard {
	if !p.Config.Platform.IsCloud() {
		return NewUnlimited()
	}

	return NewCloud(CloudConfig{
		Plans:    p.Plans,
		Counters: p.Counters,
		DB:       p.DB,
		Logger:   p.Logger,
	})
}
