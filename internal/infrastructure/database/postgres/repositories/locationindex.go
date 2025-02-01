package repositories

import (
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

type LocationIndexRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}
