package middleware

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var errNoCookie = errors.New("no session cookie")

type AuthMiddlewareParams struct {
	fx.In

	Config       *config.Config
	Service      services.AuthService
	ErrorHandler *helpers.ErrorHandler
}

type AuthMiddleware struct {
	cfg          *config.Config
	service      services.AuthService
	errorHandler *helpers.ErrorHandler
}

func NewAuthMiddleware(p AuthMiddlewareParams) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:          p.Config,
		service:      p.Service,
		errorHandler: p.ErrorHandler,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := m.authenticate(c); err != nil {
			m.errorHandler.HandleError(
				c,
				errortypes.NewAuthenticationError("Authentication required"),
			)
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) authenticate(c *gin.Context) error {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return m.authenticateWithBearer(c, strings.TrimSpace(authHeader[7:]))
	}

	return m.authenticateWithSession(c)
}

func (m *AuthMiddleware) authenticateWithSession(c *gin.Context) error {
	cookie, err := c.Cookie(m.cfg.Security.Session.Name)
	if err != nil {
		return errNoCookie
	}

	if cookie == "" {
		return errNoCookie
	}

	sessionID, err := pulid.MustParse(cookie)
	if err != nil {
		m.clearSessionCookie(c)
		return err
	}

	sess, err := m.service.ValidateSession(c.Request.Context(), sessionID)
	if err != nil {
		m.clearSessionCookie(c)
		return err
	}

	authctx.SetAuthContext(
		c,
		sess.UserID,
		sess.BusinessUnitID,
		sess.OrganizationID,
	)

	return nil
}

func (m *AuthMiddleware) authenticateWithBearer(c *gin.Context, token string) error {
	principal, err := m.service.AuthenticateAPIKey(
		c.Request.Context(),
		token,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		return err
	}

	authctx.SetAPIKeyContext(
		c,
		principal.PrincipalID,
		principal.BusinessUnitID,
		principal.OrganizationID,
	)

	return nil
}

func (m *AuthMiddleware) clearSessionCookie(c *gin.Context) {
	sessionCfg := m.cfg.Security.Session
	c.SetSameSite(sessionCfg.GetSameSite())
	c.SetCookie(
		sessionCfg.Name,
		"",
		-1,
		sessionCfg.Path,
		sessionCfg.Domain,
		sessionCfg.Secure,
		sessionCfg.HTTPOnly,
	)
}
