package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
)

func asDBConnection(conn *postgres.Connection) kernelports.DBConnection {
	return conn
}

func asTransactor(conn *postgres.Connection) kernelports.Transactor {
	return conn
}

func asSavepointer(conn *postgres.Connection) kernelports.Savepointer {
	return conn
}

var DatabaseModule = fx.Module(
	"database",
	fx.Provide(
		postgres.NewConnection,
		postgres.NewReportingConnection,
		asDBConnection,
		asTransactor,
		asSavepointer,
		postgres.NewCapabilityProbe,
		querybudget.New,
	),
	fx.Invoke(func(*postgres.CapabilityProbe) {}),
)
