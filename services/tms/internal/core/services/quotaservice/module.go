package quotaservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"go.uber.org/fx"
)

var Module = fx.Module("quota-service", fx.Provide(New))

func New() platformplanports.QuotaGuard {
	return NewUnlimited()
}
