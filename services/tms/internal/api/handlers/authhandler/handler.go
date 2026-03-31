package authhandler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service      services.AuthService
	Logger       *zap.Logger
	Config       *config.Config
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service services.AuthService
	l       *zap.Logger
	cfg     *config.Config
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		l:       p.Logger.With(zap.String("handler", "auth")),
		cfg:     p.Config,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/auth/")
	api.POST("login", h.login)
	api.POST("logout", h.logout)
	api.POST("validate-session", h.validateSession)
	api.GET("tenant/:slug", h.getTenantLoginMetadata)
	api.GET("microsoft/start/:slug", h.startSSOLogin(tenant.SSOProviderAzureAD))
	api.GET("microsoft/callback", h.ssoCallback(tenant.SSOProviderAzureAD))
	api.GET("sso/start/:provider/:slug", h.startSSOLoginGeneric)
	api.GET("sso/callback/:provider", h.ssoCallbackGeneric)
}

// @Summary Login
// @Description Authenticates a user and returns the session payload. This endpoint also sets the configured session cookie.
// @ID login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Login credentials"
// @Success 200 {object} services.LoginResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Router /auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	authctx.SetAuthContext(
		c,
		resp.User.ID,
		resp.User.BusinessUnitID,
		resp.User.CurrentOrganizationID,
	)

	h.setSessionCookie(c, resp.SessionID, resp.ExpiresAt)

	c.JSON(http.StatusOK, resp)
}

// @Summary Logout
// @Description Invalidates the current session and clears the session cookie.
// @ID logout
// @Tags Auth
// @Success 204 "No Content"
// @Failure 401 {object} helpers.ProblemDetail
// @Router /auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	sessionIDstr, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || sessionIDstr == "" {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Session not found"))
		return
	}

	sessionID, err := pulid.MustParse(sessionIDstr)
	if err != nil {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Invalid session ID"))
		return
	}

	if err = h.service.Logout(c.Request.Context(), sessionID); err != nil {
		h.l.Warn("logout failed", zap.String("sessionID", sessionID.String()), zap.Error(err))
	}

	h.clearSessionCookie(c)

	c.Status(http.StatusNoContent)
}

// @Summary Validate session
// @Description Validates the current session cookie and returns whether the session is still active.
// @ID validateSession
// @Tags Auth
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /auth/validate-session [post]
func (h *Handler) validateSession(c *gin.Context) {
	sessionIDstr, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || sessionIDstr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
		return
	}

	sessionID, err := pulid.MustParse(sessionIDstr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
		return
	}

	_, err = h.service.ValidateSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) getTenantLoginMetadata(c *gin.Context) {
	resp, err := h.service.GetTenantLoginMetadata(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) startSSOLogin(provider tenant.SSOProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		redirectURL, err := h.service.StartSSOLogin(c.Request.Context(), services.StartSSOLoginRequest{
			Provider:         provider,
			OrganizationSlug: c.Param("slug"),
			ReturnTo:         c.Query("returnTo"),
		})
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		c.Redirect(http.StatusFound, redirectURL)
	}
}

func (h *Handler) startSSOLoginGeneric(c *gin.Context) {
	provider, err := parseSSOProvider(c.Param("provider"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.startSSOLogin(provider)(c)
}

func (h *Handler) ssoCallback(provider tenant.SSOProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := strings.TrimSpace(c.Query("state"))

		var loginState *repositories.SSOLoginState
		if state != "" {
			ls, stateErr := h.service.GetSSOLoginState(c.Request.Context(), state)
			if stateErr == nil {
				loginState = ls
			}
		}

		if loginState != nil && loginState.Provider != "" && loginState.Provider != provider {
			h.redirectSSOError(
				c,
				loginState,
				errortypes.NewAuthenticationError("SSO login session does not match this provider"),
			)
			return
		}

		resp, err := h.service.HandleSSOCallback(c.Request.Context(), services.SSOCallbackRequest{
			State: state,
			Code:  strings.TrimSpace(c.Query("code")),
		})
		if err != nil {
			h.redirectSSOError(c, loginState, err)
			return
		}

		h.setSessionCookie(c, resp.LoginResponse.SessionID, resp.LoginResponse.ExpiresAt)
		c.Redirect(http.StatusFound, resp.RedirectTo)
	}
}

func (h *Handler) ssoCallbackGeneric(c *gin.Context) {
	provider, err := parseSSOProvider(c.Param("provider"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.ssoCallback(provider)(c)
}

func parseSSOProvider(raw string) (tenant.SSOProvider, error) {
	switch strings.ToLower(raw) {
	case "microsoft", "azuread":
		return tenant.SSOProviderAzureAD, nil
	case "okta":
		return tenant.SSOProviderOkta, nil
	default:
		return "", errortypes.NewValidationError("provider", errortypes.ErrInvalid, "Unsupported SSO provider")
	}
}

func (h *Handler) redirectSSOError(c *gin.Context, loginState *repositories.SSOLoginState, callbackErr error) {
	h.l.Warn("sso callback failed", zap.Error(callbackErr))

	loginPath := "/login"
	if loginState != nil && loginState.OrganizationSlug != "" {
		loginPath = "/login/" + loginState.OrganizationSlug
	}

	userMsg := "Sign-in failed. Please try again or contact your administrator."

	origin := h.cfg.Server.CORS.AllowedOrigins[0]
	if loginState != nil {
		if returnURL, err := url.Parse(loginState.ReturnTo); err == nil && returnURL.Scheme != "" && returnURL.Host != "" {
			origin = returnURL.Scheme + "://" + returnURL.Host
		}
	}
	redirectURL := fmt.Sprintf("%s%s?sso_error=%s", origin, loginPath, url.QueryEscape(userMsg))
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	sessionCfg := h.cfg.Security.Session
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

func (h *Handler) setSessionCookie(c *gin.Context, sessionID string, expiresAt int64) {
	maxAge := max(0, int(expiresAt-timeutils.NowUnix()))

	sessionCfg := h.cfg.Security.Session
	c.SetSameSite(sessionCfg.GetSameSite())
	c.SetCookie(
		sessionCfg.Name,
		sessionID,
		maxAge,
		sessionCfg.Path,
		sessionCfg.Domain,
		sessionCfg.Secure,
		sessionCfg.HTTPOnly,
	)
}
