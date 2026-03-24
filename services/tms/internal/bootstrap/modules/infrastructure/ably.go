package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/ably"
	"go.uber.org/fx"
)

var AblyClientModule = fx.Module("ably-client", fx.Provide(ably.New))
