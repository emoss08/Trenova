package helpers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestErrorHandler(debug bool) *helpers.ErrorHandler {
	logger := zap.NewNop()
	cfg := &config.Config{
		App: config.AppConfig{
			Debug:              debug,
			ProblemTypeBaseURI: "https://api.test.com/problems/",
		},
	}

	return helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})
}

func TestNewErrorHandler(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	assert.NotNil(t, handler)
}

func TestErrorHandler_HandleError_NilError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext()

	handler.HandleError(ctx.Context, nil)

	assert.Equal(t, http.StatusOK, ctx.ResponseCode())
}

func TestErrorHandler_HandleError_ValidationError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext().WithPath("/api/users")

	me := errortypes.NewMultiError()
	me.Add("email", errortypes.ErrRequired, "Email is required")
	me.Add("password", errortypes.ErrInvalidLength, "Password too short")

	handler.HandleError(ctx.Context, me)

	assert.Equal(t, http.StatusBadRequest, ctx.ResponseCode())
	assert.Equal(t, helpers.ProblemJSONContentType, ctx.ResponseHeader("Content-Type"))

	var problem helpers.ProblemDetail
	err := ctx.ResponseJSON(&problem)
	require.NoError(t, err)

	assert.Equal(t, "https://api.test.com/problems/validation-error", problem.Type)
	assert.Equal(t, "Validation Failed", problem.Title)
	assert.Equal(t, http.StatusBadRequest, problem.Status)
	assert.Len(t, problem.Errors, 2)
}

func TestErrorHandler_HandleError_BusinessError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext().WithPath("/api/orders")

	err := errortypes.NewBusinessError("Insufficient inventory")

	handler.HandleError(ctx.Context, err)

	assert.Equal(t, http.StatusUnprocessableEntity, ctx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ctx.ResponseJSON(&problem))

	assert.Equal(t, "https://api.test.com/problems/business-rule-violation", problem.Type)
	assert.Equal(t, "Business Rule Violation", problem.Title)
	assert.Contains(t, problem.Detail, "Insufficient inventory")
}

func TestErrorHandler_HandleError_NotFoundError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext().WithPath("/api/users/123")

	err := errortypes.NewNotFoundError("User not found")

	handler.HandleError(ctx.Context, err)

	assert.Equal(t, http.StatusNotFound, ctx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ctx.ResponseJSON(&problem))

	assert.Equal(t, "https://api.test.com/problems/resource-not-found", problem.Type)
}

func TestErrorHandler_HandleError_AuthenticationError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext()

	err := errortypes.NewAuthenticationError("Invalid token")

	handler.HandleError(ctx.Context, err)

	assert.Equal(t, http.StatusUnauthorized, ctx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ctx.ResponseJSON(&problem))

	assert.Equal(t, "https://api.test.com/problems/authentication-error", problem.Type)
}

func TestErrorHandler_HandleError_AuthorizationError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext()

	err := errortypes.NewAuthorizationError("Access denied")

	handler.HandleError(ctx.Context, err)

	assert.Equal(t, http.StatusForbidden, ctx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ctx.ResponseJSON(&problem))

	assert.Equal(t, "https://api.test.com/problems/authorization-error", problem.Type)
}

func TestErrorHandler_HandleError_RateLimitError(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext()

	err := errortypes.NewRateLimitError("api", "Too many requests")

	handler.HandleError(ctx.Context, err)

	assert.Equal(t, http.StatusTooManyRequests, ctx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ctx.ResponseJSON(&problem))

	assert.Equal(t, "https://api.test.com/problems/rate-limit-exceeded", problem.Type)
}

func TestErrorHandler_HandleError_DatabaseError(t *testing.T) {
	t.Parallel()

	t.Run("hides details in production", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)
		ctx := testutil.NewGinTestContext()

		err := errortypes.NewDatabaseError("SQLSTATE 42P01: relation does not exist")

		handler.HandleError(ctx.Context, err)

		assert.Equal(t, http.StatusInternalServerError, ctx.ResponseCode())

		var problem helpers.ProblemDetail
		require.NoError(t, ctx.ResponseJSON(&problem))

		assert.NotContains(t, problem.Detail, "SQLSTATE")
		assert.Contains(t, problem.Detail, "unexpected error")
	})

	t.Run("shows details in debug mode", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(true)
		ctx := testutil.NewGinTestContext()

		err := errortypes.NewDatabaseError("SQLSTATE 42P01: relation does not exist")

		handler.HandleError(ctx.Context, err)

		var problem helpers.ProblemDetail
		require.NoError(t, ctx.ResponseJSON(&problem))

		assert.Contains(t, problem.Detail, "SQLSTATE")
	})
}

func TestErrorHandler_HandleError_InternalError(t *testing.T) {
	t.Parallel()

	t.Run("hides details in production", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)
		ctx := testutil.NewGinTestContext()

		err := errors.New("panic: nil pointer dereference")

		handler.HandleError(ctx.Context, err)

		assert.Equal(t, http.StatusInternalServerError, ctx.ResponseCode())

		var problem helpers.ProblemDetail
		require.NoError(t, ctx.ResponseJSON(&problem))

		assert.NotContains(t, problem.Detail, "nil pointer")
		assert.Contains(t, problem.Detail, "unexpected error")
	})
}

func TestErrorHandler_HandleError_RequestID(t *testing.T) {
	t.Parallel()

	t.Run("includes X-Request-ID header in instance", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)
		ctx := testutil.NewGinTestContext().
			WithPath("/api/test").
			WithHeader("X-Request-ID", "req-12345")

		handler.HandleError(ctx.Context, errors.New("test error"))

		var problem helpers.ProblemDetail
		require.NoError(t, ctx.ResponseJSON(&problem))

		assert.Equal(t, "/api/test#req-12345", problem.Instance)
		assert.Equal(t, "req-12345", problem.TraceID)
	})

	t.Run("uses context request_id when header missing", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)
		ctx := testutil.NewGinTestContext().
			WithPath("/api/test").
			WithContextValue("request_id", "ctx-67890")

		handler.HandleError(ctx.Context, errors.New("test error"))

		var problem helpers.ProblemDetail
		require.NoError(t, ctx.ResponseJSON(&problem))

		assert.Equal(t, "/api/test#ctx-67890", problem.Instance)
		assert.Equal(t, "ctx-67890", problem.TraceID)
	})
}

func TestErrorHandler_HandleError_AbortsContext(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext()

	handler.HandleError(ctx.Context, errors.New("test"))

	assert.True(t, ctx.Context.IsAborted())
}

func TestErrorHandler_Middleware(t *testing.T) {
	t.Parallel()

	t.Run("handles panics", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(handler.Middleware())
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, helpers.ProblemJSONContentType, w.Header().Get("Content-Type"))

		var problem helpers.ProblemDetail
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "https://api.test.com/problems/internal-error", problem.Type)
	})

	t.Run("handles c.Error() errors", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(handler.Middleware())
		router.GET("/error", func(c *gin.Context) {
			_ = c.Error(errortypes.NewNotFoundError("resource not found"))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var problem helpers.ProblemDetail
		err := json.Unmarshal(w.Body.Bytes(), &problem)
		require.NoError(t, err)

		assert.Equal(t, "https://api.test.com/problems/resource-not-found", problem.Type)
	})

	t.Run("passes through successful requests", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(handler.Middleware())
		router.GET("/success", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handles error panics", func(t *testing.T) {
		t.Parallel()
		handler := newTestErrorHandler(false)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(handler.Middleware())
		router.GET("/error-panic", func(c *gin.Context) {
			panic(errors.New("error panic"))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/error-panic", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestErrorHandler_Middleware_MultipleErrors(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(handler.Middleware())
	router.GET("/multi-error", func(c *gin.Context) {
		_ = c.Error(errors.New("first error"))
		_ = c.Error(errortypes.NewNotFoundError("second error"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/multi-error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
