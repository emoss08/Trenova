package email

import (
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"go.uber.org/fx"
)

var Module = fx.Module("email",
	providers.Module,
	fx.Provide(NewService),
)
