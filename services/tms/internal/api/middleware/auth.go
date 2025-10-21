package middleware

import (
	"net/http"
	"strings"

	authcontext "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AuthMiddlewareParams struct {
	fx.In

	Config      *config.Config
	AuthService services.AuthService
	Logger      *zap.Logger
}

type AuthMiddleware struct {
	cfg         *config.Config
	authService services.AuthService
	l           *zap.Logger
}

func NewAuthMiddleware(p AuthMiddlewareParams) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:         p.Config,
		authService: p.AuthService,
		l:           p.Logger.With(zap.String("middleware", "auth")),
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if token := m.extractBearerToken(c); token != "" {
			if err := m.authenticateWithToken(c, token); err == nil {
				c.Next()
				return
			}
		}

		if err := m.authenticateWithSession(c); err != nil {
			m.handleAuthError(c, http.StatusUnauthorized, "Authentication required")
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := m.authenticateWithSession(c); err != nil {
			m.handleAuthError(c, http.StatusUnauthorized, "Session authentication required")
			return
		}
		c.Next()
	}
}

func (m *AuthMiddleware) RequireAPIToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractBearerToken(c)
		if token == "" {
			m.handleAuthError(c, http.StatusUnauthorized, "Bearer token required")
			return
		}

		if err := m.authenticateWithToken(c, token); err != nil {
			m.handleAuthError(c, http.StatusUnauthorized, "Invalid API token")
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireScopes(scopes ...tenant.APITokenScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.isAuthenticated(c) {
			m.handleAuthError(c, http.StatusUnauthorized, "Authentication required")
			return
		}

		if authcontext.IsSessionAuth(c) {
			c.Next()
			return
		}

		token, exists := authcontext.GetAPIToken(c)
		if !exists {
			m.handleAuthError(c, http.StatusForbidden, "API token required for this operation")
			return
		}

		if !token.HasAllScopes(scopes...) {
			m.handleAuthError(c, http.StatusForbidden, "Insufficient permissions")
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if token := m.extractBearerToken(c); token != "" {
			_ = m.authenticateWithToken(c, token) // Ignore error
		} else {
			_ = m.authenticateWithSession(c) // Ignore error
		}

		c.Next()
	}
}

func (m *AuthMiddleware) authenticateWithSession(c *gin.Context) error {
	cookie, err := c.Cookie(m.cfg.Security.Session.Name)
	if err != nil || cookie == "" {
		return err
	}

	sessionID, err := pulid.Parse(cookie)
	if err != nil {
		return err
	}

	sess, err := m.authService.RefreshSession(c.Request.Context(), services.RefreshSessionRequest{
		SessionID: sessionID,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		m.l.Info(
			"Session authentication failed",
			zap.Error(err),
			zap.Any("context", c.Request.Context()),
		)
		m.clearSessionCookie(c)
		return err
	}

	authcontext.SetAuthContext(
		c,
		sess.UserID,
		sess.BusinessUnitID,
		sess.OrganizationID,
		authcontext.AuthTypeSession,
	)
	authcontext.SetSession(c, sess)

	m.l.Debug("Session authenticated",
		zap.String("sessionId", sess.ID.String()),
		zap.String("userId", sess.UserID.String()),
	)

	return nil
}

func (m *AuthMiddleware) authenticateWithToken(c *gin.Context, token string) error {
	apiToken, err := m.authService.ValidateAPIToken(
		c.Request.Context(),
		services.ValidateAPITokenRequest{
			Token:    token,
			ClientIP: c.ClientIP(),
		},
	)
	if err != nil {
		return err
	}

	authcontext.SetAuthContext(
		c,
		apiToken.UserID,
		apiToken.BusinessUnitID,
		apiToken.OrganizationID,
		authcontext.AuthTypeAPIToken,
	)
	authcontext.SetAPIToken(c, apiToken)

	m.l.Debug("API token authenticated",
		zap.String("tokenId", apiToken.ID.String()),
		zap.String("userId", apiToken.UserID.String()),
		zap.String("tokenName", apiToken.Name),
	)

	return nil
}

func (m *AuthMiddleware) extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimPrefix(auth, prefix)
}

func (m *AuthMiddleware) isAuthenticated(c *gin.Context) bool {
	_, exists := authcontext.GetUserID(c)
	return exists
}

func (m *AuthMiddleware) handleAuthError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  status,
	})
	c.Abort()
}

func (m *AuthMiddleware) clearSessionCookie(c *gin.Context) {
	c.SetCookie(
		m.cfg.Security.Session.Name,
		"",
		-1,
		m.cfg.Security.Session.Path,
		m.cfg.Security.Session.Domain,
		m.cfg.Security.Session.Secure,
		m.cfg.Security.Session.HTTPOnly,
	)
}
