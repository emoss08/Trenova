package middleware

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var errNoCookie = errors.New("no session cookie")

type AuthMiddlewareParams struct {
	fx.In

	SecurityConfig *securityconfig.Config
	Service        tenantports.AuthService
	ErrorHandler   *helpers.ErrorHandler
}

type AuthMiddleware struct {
	cfg          *securityconfig.Config
	service      tenantports.AuthService
	errorHandler *helpers.ErrorHandler
}

func NewAuthMiddleware(p AuthMiddlewareParams) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:          p.SecurityConfig,
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
	cookie, err := c.Cookie(m.cfg.Session.Name)
	if err != nil {
		return errNoCookie
	}

	if cookie == "" {
		return errNoCookie
	}

	sess, err := m.service.AuthenticateSession(c.Request.Context(), cookie)
	if err != nil {
		m.clearSessionCookie(c)
		return err
	}

	authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
		SessionID:              sess.ID,
		UserID:                 sess.UserID,
		BusinessUnitID:         sess.BusinessUnitID,
		OrganizationID:         sess.OrganizationID,
		ActiveRoleIDs:          sess.ActiveRoleIDs,
		RequiresRoleActivation: sess.RoleActivationRequired && len(sess.ActiveRoleIDs) == 0,
		RoleActivation:         sess.RoleActivationRequired,
		AuthProvider:           sess.AuthProvider,
		ExternalIdentityID:     sess.ExternalIdentityID,
		ExternalSubject:        sess.ExternalSubject,
		AuthenticatorAAL:       sess.AuthenticatorAAL,
		FederationFAL:          sess.FederationFAL,
		MFAAuthenticatedAt:     sess.MFAAuthenticatedAt,
		LastReauthenticatedAt:  sess.LastReauthenticatedAt,
		RiskDecision:           sess.RiskDecision,
		RiskDecisionID:         sess.RiskDecisionID,
		IsPortalUser:           sess.IsPortalUser,
		MustChangePassword:     sess.MustChangePassword,
		Locale:                 sess.Locale,
	})

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
	sessionCfg := m.cfg.Session
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
