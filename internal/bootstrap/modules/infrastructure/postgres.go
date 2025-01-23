package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres"
	"go.uber.org/fx"
)

var DatabaseModule = fx.Module("db", fx.Provide(postgres.NewConnection))
