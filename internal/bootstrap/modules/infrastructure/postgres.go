package infrastructure

import (
	"github.com/trenova-app/transport/internal/infrastructure/database/postgres"
	"go.uber.org/fx"
)

var DatabaseModule = fx.Module("db", fx.Provide(postgres.NewConnection))
