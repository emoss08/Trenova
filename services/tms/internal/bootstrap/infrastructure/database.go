package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
)

func asDBConnection(conn *postgres.Connection) kernelports.DBConnection {
	return conn
}

var DatabaseModule = fx.Module(
	"database",
	fx.Provide(
		postgres.NewConnection,
		postgres.NewReportingConnection,
		asDBConnection,
		postgres.NewCapabilityProbe,
	),
	fx.Invoke(func(*postgres.CapabilityProbe) {}),
)
