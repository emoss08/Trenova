package middleware

import (
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const eventStreamMediaType = "text/event-stream"

type QueryBudgetMiddlewareParams struct {
	fx.In

	Budgets *querybudget.Budgets
}

type QueryBudgetMiddleware struct {
	budgets *querybudget.Budgets
}

func NewQueryBudgetMiddleware(p QueryBudgetMiddlewareParams) *QueryBudgetMiddleware {
	return &QueryBudgetMiddleware{budgets: p.Budgets}
}

func (m *QueryBudgetMiddleware) Install() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil && !longLived(c) {
			c.Request = c.Request.WithContext(m.budgets.Start(
				c.Request.Context(),
				zap.String("method", c.Request.Method),
				zap.String("route", c.FullPath()),
			))
		}

		c.Next()
	}
}

func longLived(c *gin.Context) bool {
	return c.IsWebsocket() ||
		strings.Contains(c.GetHeader("Accept"), eventStreamMediaType)
}
