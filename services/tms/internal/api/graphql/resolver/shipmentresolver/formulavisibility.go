package shipmentresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func mayReadFormulas(ctx context.Context, core *base.Core) (bool, error) {
	authCtx, err := core.RequireAuth(ctx)
	if err != nil {
		return false, err
	}

	return core.HasPermission(
		ctx,
		authCtx,
		permission.ResourceFormulaTemplate,
		permission.OpRead,
	), nil
}
