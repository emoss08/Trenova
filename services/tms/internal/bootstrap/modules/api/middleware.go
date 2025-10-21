package api

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"go.uber.org/fx"
)

var MiddlewareModule = fx.Module(
	"api-middleware",
	fx.Provide(
		middleware.NewAuthMiddleware,
		middleware.NewPermissionMiddleware,
	),
)
