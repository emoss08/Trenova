package fieldguard

import (
	"context"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
)

func Visible(c *gin.Context, engine permissionports.PermissionEngine) permission.FieldGroupSet {
	authCtx := authctx.GetAuthContext(c)
	if authCtx == nil || engine == nil {
		return 0
	}

	return fieldsensitivity.VisibleGroups(
		c.Request.Context(),
		func(ctx context.Context, resource permission.Resource) (bool, error) {
			result, err := engine.Check(
				ctx,
				middleware.BuildPermissionCheckRequest(authCtx, resource.String(), permission.OpRead),
			)
			if err != nil || result == nil {
				return false, err
			}
			return result.Allowed, nil
		},
	)
}

func Withhold[T permission.FieldWithholder](visible permission.FieldGroupSet, items ...T) {
	if visible.HasAll() {
		return
	}
	for _, item := range items {
		item.WithholdFields(visible)
	}
}
