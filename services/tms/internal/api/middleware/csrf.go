package middleware

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

type CSRFMiddleware struct {
	cfg          *config.Config
	errorHandler *helpers.ErrorHandler
}

func NewCSRFMiddleware(cfg *config.Config, errorHandler *helpers.ErrorHandler) *CSRFMiddleware {
	return &CSRFMiddleware{
		cfg:          cfg,
		errorHandler: errorHandler,
	}
}

func (m *CSRFMiddleware) RequireToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		authCtx := authctx.GetAuthContext(c)
		if authCtx == nil || authCtx.PrincipalType != authctx.PrincipalTypeUser {
			c.Next()
			return
		}

		sessionID, err := c.Cookie(m.cfg.Security.Session.Name)
		if err != nil || sessionID == "" {
			m.reject(c)
			return
		}

		token := c.GetHeader(m.cfg.Security.CSRF.HeaderName)
		if token == "" || !csrf.Verify(token, sessionID, m.cfg.Security.Session.Secret) {
			m.reject(c)
			return
		}

		c.Next()
	}
}

func (m *CSRFMiddleware) reject(c *gin.Context) {
	m.errorHandler.HandleError(
		c,
		errortypes.NewAuthorizationError("CSRF token required"),
	)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
