package graphql

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
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
		metrics: metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
	}

	router := gin.New()
	router.POST("/graphql", func(c *gin.Context) {
		authctx.SetAPIKeyContext(
			c,
			pulid.MustNew("ak_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		h.handle(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "API keys cannot access GraphQL")
}

func TestHandler_RejectsOversizedBodyByContentLength(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Debug:              true,
			ProblemTypeBaseURI: "https://api.test/problems/",
		},
		Security: config.SecurityConfig{
			GraphQL: config.GraphQLSecurityConfig{MaxRequestBodyBytes: 2048},
		},
	}
	h := &Handler{
		cfg: cfg,
		eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
		metrics: metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
	}

	router := gin.New()
	router.POST("/graphql", func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		h.handle(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/graphql",
		strings.NewReader(strings.Repeat("x", 4096)),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), requestTooLargeErrorCode)
	assert.Contains(t, w.Body.String(), "2048 byte limit")
}

func TestHandler_RejectsOversizedBodyWithoutContentLength(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Debug:              true,
			ProblemTypeBaseURI: "https://api.test/problems/",
		},
		Security: config.SecurityConfig{
			GraphQL: config.GraphQLSecurityConfig{MaxRequestBodyBytes: 1024},
		},
	}
	h := &Handler{
		cfg: cfg,
		eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
		persistedOps: &PersistedOperationManifest{queries: map[string]string{}},
		metrics:      metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
	}

	router := gin.New()
	router.POST("/graphql", func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		h.handle(c)
	})

	w := httptest.NewRecorder()
	body := `{"query":"` + strings.Repeat("a", 2048) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), requestTooLargeErrorCode)
}

func TestStatusOverrideWriter_RewritesProtocolStatus(t *testing.T) {
	t.Parallel()

	status := gqlctx.NewResponseStatus()
	status.Override(http.StatusTooManyRequests, 1500*time.Millisecond)

	recorder := httptest.NewRecorder()
	writer := statusOverrideWriter{ResponseWriter: recorder, status: status}
	writer.WriteHeader(http.StatusUnprocessableEntity)

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Equal(t, "2", recorder.Header().Get("Retry-After"))
}

func TestStatusOverrideWriter_LeavesOtherStatusesAlone(t *testing.T) {
	t.Parallel()

	status := gqlctx.NewResponseStatus()
	status.Override(http.StatusTooManyRequests, time.Second)

	recorder := httptest.NewRecorder()
	writer := statusOverrideWriter{ResponseWriter: recorder, status: status}
	writer.WriteHeader(http.StatusOK)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get("Retry-After"))

	untouched := httptest.NewRecorder()
	statusOverrideWriter{ResponseWriter: untouched, status: gqlctx.NewResponseStatus()}.
		WriteHeader(http.StatusUnprocessableEntity)
	assert.Equal(t, http.StatusUnprocessableEntity, untouched.Code)
}
