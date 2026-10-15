package middleware

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ReadScopeMiddlewareParams struct {
	fx.In

	Scopes permissionports.ReadScopes
}

type ReadScopeMiddleware struct {
	scopes permissionports.ReadScopes
}

func NewReadScopeMiddleware(p ReadScopeMiddlewareParams) *ReadScopeMiddleware {
	return &ReadScopeMiddleware{scopes: p.Scopes}
}

func (m *ReadScopeMiddleware) Install() gin.HandlerFunc {
	return func(c *gin.Context) {
		if authCtx := authctx.GetAuthContext(c); authCtx != nil && c.Request != nil {
			principal := BuildPermissionCheckRequest(authCtx, "", "")
			ctx := readscope.With(c.Request.Context(), m.scopes.Resolver(principal))
			c.Request = c.Request.WithContext(ctx)
			c.Set(readscope.GinContextKey, readscope.Current(ctx))
		}

		c.Next()
	}
}
