package tableconfigurationresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
)

func (d *Deps) mayReadTable(ctx context.Context, authCtx *authctx.AuthContext, resource string) bool {
	if resource == "" {
		return false
	}
	return d.HasPermission(ctx, authCtx, permission.Resource(resource), permission.OpRead)
}

func (d *Deps) readableTableResources(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource string,
) ([]string, error) {
	if resource != "" {
		if d.mayReadTable(ctx, authCtx, resource) {
			return []string{resource}, nil
		}
		return []string{}, nil
	}

	definitions := fieldsensitivity.DefaultRegistry().All()
	readable := make([]string, 0, len(definitions))
	for _, def := range definitions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if d.mayReadTable(ctx, authCtx, def.Resource) {
			readable = append(readable, def.Resource)
		}
	}
	return readable, nil
}
