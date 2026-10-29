package edihandlertest

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type Routes interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

type Middleware struct {
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

func NewMiddleware(logger *zap.Logger) Middleware {
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &appconfig.Config{Debug: true},
	})

	return Middleware{
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
	}
}

func Run(
	t *testing.T,
	handler Routes,
	method string,
	path string,
	query map[string]string,
	body any,
	wantStatus int,
) *testutil.GinTestContext {
	t.Helper()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithDefaultAuthContext()
	if len(query) > 0 {
		ginCtx.WithQuery(query)
	}
	if body != nil {
		ginCtx.WithJSONBody(body)
	}
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, wantStatus, ginCtx.ResponseCode())
	return ginCtx
}
