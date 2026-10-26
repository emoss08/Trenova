package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func budgetRouter(t *testing.T) (*gin.Engine, *bool) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	budgets := querybudget.New(querybudget.Params{Config: &config.Config{}, Logger: zap.NewNop()})
	mw := NewQueryBudgetMiddleware(QueryBudgetMiddlewareParams{Budgets: budgets})

	var budgeted bool
	router := gin.New()
	router.Use(mw.Install())
	router.GET("/api/v1/shipments/:id/", func(c *gin.Context) {
		_, budgeted = keel.BudgetReportFrom(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	return router, &budgeted
}

func TestQueryBudgetMiddlewareBudgetsEachRequest(t *testing.T) {
	router, budgeted := budgetRouter(t)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/shipments/shp_1/",
		nil,
	)
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.True(t, *budgeted)
}

func TestQueryBudgetMiddlewareLeavesEventStreamsAlone(t *testing.T) {
	router, budgeted := budgetRouter(t)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/shipments/shp_1/",
		nil,
	)
	req.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, *budgeted, "a stream lives for minutes and repeats its reads by design")
}
