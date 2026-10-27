package carrierintelrepository

import (
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

func saveError(m dberror.Mapping, err error, id, op string) error {
	if dberror.IsVersionConflict(err) || dberror.IsNotFoundError(err) {
		return m.Map(err, id)
	}

	return fmt.Errorf("%s: %w", op, err)
}
