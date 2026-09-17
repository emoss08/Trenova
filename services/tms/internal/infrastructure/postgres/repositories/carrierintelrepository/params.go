package carrierintelrepository

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}
