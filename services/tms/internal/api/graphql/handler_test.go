package graphql

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandler_RejectsAPIKeyPrincipal(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Debug:              true,
			ProblemTypeBaseURI: "https://api.test/problems/",
		},
	}
	h := &Handler{
		eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
	}

	router := gin.New()
	router.POST("/graphql", func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, pulid.MustNew("ak_"), pulid.MustNew("bu_"), pulid.MustNew("org_"))
		h.handle(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "API keys cannot access GraphQL")
}

func TestHandler_PlaygroundEnabledForDevelopment(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:   config.EnvDevelopment,
				Debug: true,
			},
		},
	}

	router := gin.New()
	router.GET("/graphql", h.handlePlayground)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Trenova GraphQL")
	assert.Contains(t, w.Body.String(), `const graphqlEndpoint = "/graphql";`)
	assert.Contains(t, w.Body.String(), `const csrfEndpoint = "/api/v1/auth/csrf";`)
	assert.Contains(t, w.Body.String(), `credentials: "include"`)
	assert.Equal(t, playgroundContentSecurityPolicy, w.Header().Get("Content-Security-Policy"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "https://cdn.jsdelivr.net")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "'unsafe-inline'")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "connect-src 'self' https://cdn.jsdelivr.net")
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestHandler_PlaygroundDisabledForProduction(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg: &config.Config{
			App: config.AppConfig{
				Env:   config.EnvProduction,
				Debug: false,
			},
		},
	}

	router := gin.New()
	router.GET("/graphql", h.handlePlayground)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, w.Body.String())
}
