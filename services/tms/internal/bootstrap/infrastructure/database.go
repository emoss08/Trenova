package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
)

func asDBConnection(conn *postgres.Connection) ports.DBConnection {
	return conn
}

var DatabaseModule = fx.Module("database", fx.Provide(postgres.NewConnection, asDBConnection))
