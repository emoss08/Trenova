package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/gotenberg"
	"go.uber.org/fx"
)

var GotenbergModule = fx.Module("gotenberg",
	fx.Provide(gotenberg.NewClient),
)
