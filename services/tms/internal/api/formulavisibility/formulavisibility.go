package formulavisibility

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
)

func Apply[T any](
	c *gin.Context,
	pm *middleware.PermissionMiddleware,
	value T,
	redact func(T) T,
) (T, error) {
	allowed, err := pm.Allowed(
		c,
		permission.ResourceFormulaTemplate.String(),
		permission.OpRead,
	)
	if err != nil {
		var zero T
		return zero, err
	}
	if allowed {
		return value, nil
	}

	return redact(value), nil
}

func Respond[T any](
	c *gin.Context,
	pm *middleware.PermissionMiddleware,
	eh *helpers.ErrorHandler,
	status int,
	value T,
	redact func(T) T,
) {
	visible, err := Apply(c, pm, value, redact)
	if err != nil {
		eh.HandleError(c, err)
		return
	}

	c.JSON(status, visible)
}
