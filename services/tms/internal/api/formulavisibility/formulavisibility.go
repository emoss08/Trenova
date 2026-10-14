package formulavisibility

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Gate struct {
	permissions *middleware.PermissionMiddleware
	logger      *zap.Logger
}

func NewGate(permissions *middleware.PermissionMiddleware, logger *zap.Logger) Gate {
	return Gate{permissions: permissions, logger: logger.Named("formula-visibility")}
}

func (g Gate) MayRead(c *gin.Context) bool {
	allowed, err := g.permissions.Allowed(
		c,
		permission.ResourceFormulaTemplate.String(),
		permission.OpRead,
	)
	if err != nil {
		g.logger.Warn("formula template permission check failed; hiding formula content",
			zap.String("path", c.FullPath()),
			zap.Error(err))
		return false
	}

	return allowed
}

func Apply[T any](c *gin.Context, gate Gate, value T, redact func(T) T) T {
	if gate.MayRead(c) {
		return value
	}

	return redact(value)
}

func Respond[T any](c *gin.Context, gate Gate, status int, value T, redact func(T) T) {
	c.JSON(status, Apply(c, gate, value, redact))
}
