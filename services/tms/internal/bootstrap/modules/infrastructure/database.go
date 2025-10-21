package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
)

var DatabaseModule = fx.Module("database",
	fx.Provide(postgres.NewConnection),
)
