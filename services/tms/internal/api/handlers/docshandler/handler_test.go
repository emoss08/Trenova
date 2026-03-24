package docshandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/docshandler"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler_Reference(t *testing.T) {
	t.Parallel()

	handler := docshandler.New(docshandler.Params{
		Config: &config.Config{
			App: config.AppConfig{
				Version: "test-version",
			},
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/reference")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Contains(t, ginCtx.Recorder.Body.String(), "/api/v1/openapi/openapi-3.json")
	assert.Contains(t, ginCtx.Recorder.Body.String(), "@scalar/api-reference")
}

func TestDocsHandler_Spec(t *testing.T) {
	t.Parallel()

	handler := docshandler.New(docshandler.Params{
		Config: &config.Config{
			App: config.AppConfig{
				Version: "test-version",
			},
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/openapi/swagger.json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Equal(t, "application/json; charset=utf-8", ginCtx.Recorder.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, "2.0", body["swagger"])
	assert.NotEmpty(t, body["paths"])
}

func TestDocsHandler_OpenAPI3Spec(t *testing.T) {
	t.Parallel()

	handler := docshandler.New(docshandler.Params{
		Config: &config.Config{
			App: config.AppConfig{
				Version: "test-version",
			},
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/openapi/openapi-3.json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Equal(t, "application/json; charset=utf-8", ginCtx.Recorder.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, "3.0.3", body["openapi"])
	assert.NotEmpty(t, body["paths"])
	assert.NotEmpty(t, body["servers"])
	assert.NotEmpty(t, body["x-tagGroups"])
}
