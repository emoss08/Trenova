//revive:disable-next-line:var-naming
package api

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"go.uber.org/fx"
)

var MiddlewareModule = fx.Module("api-middleware", fx.Provide(
	middleware.NewAuthMiddleware,
	middleware.NewReadScopeMiddleware,
	middleware.NewQueryBudgetMiddleware,
	middleware.NewPermissionMiddleware,
	middleware.NewEntitlementMiddleware,
	middleware.NewReadOnlyGuard,
	middleware.NewRateLimiter,
	middleware.NewIdempotencyMiddleware,
	middleware.NewTenantBoundaryMiddleware,
	responseguard.NewGuard,
))
