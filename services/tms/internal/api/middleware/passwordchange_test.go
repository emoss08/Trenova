package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestIsPasswordChangeExempt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path   string
		exempt bool
	}{
		{"/api/v1/users/me/change-password/", true},
		// The router registers some of these with a trailing slash and some without,
		// so the comparison has to normalise rather than match literally.
		{"/api/v1/users/me/change-password", true},
		{"/api/v1/users/me/", true},
		{"/api/v1/auth/logout", true},
		{"/api/v1/auth/csrf", true},
		{"/api/v1/shipments/", false},
		{"/api/v1/users/", false},
		// A path that merely starts with an exempt one must not slip through.
		{"/api/v1/users/me/profile-picture/", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.exempt, isPasswordChangeExempt(tt.path))
		})
	}
}

func TestMustChangePassword_ReadsTheSessionFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		set      bool
		value    any
		expected bool
	}{
		{"flag absent", false, nil, false},
		{"flag false", true, false, false},
		{"flag true", true, true, true},
		// A value of the wrong type must fail closed to "no demand" rather than panic.
		{"flag not a bool", true, "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.set {
				c.Set(string(authctx.MustChangePasswordKey), tt.value)
			}

			assert.Equal(t, tt.expected, mustChangePassword(c))
		})
	}
}

// The client has a step for this, but a step is a suggestion: the session cookie alone
// must not get past the API.
func TestRequireCurrentPassword_BlocksAnUnrelatedRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{},
	})

	reached := false
	engine.Use(func(ctx *gin.Context) {
		ctx.Set(string(authctx.MustChangePasswordKey), true)
		ctx.Next()
	})
	engine.Use(NewPasswordChangeMiddleware(errorHandler).RequireCurrentPassword())
	engine.GET("/api/v1/shipments/", func(ctx *gin.Context) {
		reached = true
		ctx.Status(http.StatusOK)
	})

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/shipments/", nil))

	assert.False(t, reached, "the handler must not run")
	assert.NotEqual(t, http.StatusOK, recorder.Code)
}

// The endpoint that resolves the demand has to stay reachable, or the session is
// wedged with no way out.
func TestRequireCurrentPassword_AllowsTheChangePasswordEndpoint(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{},
	})

	reached := false
	engine.Use(func(ctx *gin.Context) {
		ctx.Set(string(authctx.MustChangePasswordKey), true)
		ctx.Next()
	})
	engine.Use(NewPasswordChangeMiddleware(errorHandler).RequireCurrentPassword())
	engine.POST("/api/v1/users/me/change-password/", func(ctx *gin.Context) {
		reached = true
		ctx.Status(http.StatusOK)
	})

	engine.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/users/me/change-password/", nil),
	)

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// A session with nothing owing must pass straight through.
func TestRequireCurrentPassword_LetsACleanSessionThrough(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{},
	})

	reached := false
	engine.Use(NewPasswordChangeMiddleware(errorHandler).RequireCurrentPassword())
	engine.GET("/api/v1/shipments/", func(ctx *gin.Context) {
		reached = true
		ctx.Status(http.StatusOK)
	})

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/shipments/", nil))

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, recorder.Code)
}
