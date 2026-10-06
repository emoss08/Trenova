package quotaservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("quota-service", fx.Provide(New))

func New() services.QuotaGuard {
	return NewUnlimited()
}
