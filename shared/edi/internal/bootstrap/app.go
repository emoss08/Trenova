package bootstrap

import (
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/database"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/logging"
	"go.uber.org/fx"
)

var Module = fx.Module("bootstrap",
	logging.Module,
	fx.Provide(
		database.NewConfig,
		database.NewDatabase,
	),
	fx.Invoke(
		database.RunMigrations,
	),
)