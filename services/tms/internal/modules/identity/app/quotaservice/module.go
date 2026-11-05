package quotaservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

var Module = fx.Module("quota-service", fx.Provide(New))

type Params struct {
	fx.In

	Config *managedconfig.Config
	Gate   managedusageports.Gate
	Clock  timeutils.Clock
}

func New(p Params) platformplanports.QuotaGuard {
	if p.Config != nil && p.Config.Enabled {
		return NewManaged(p.Gate, p.Clock)
	}
	return NewUnlimited()
}
