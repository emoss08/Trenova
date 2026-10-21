package authhandler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/sessioncookie"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// validKey says whether the session a request carries is still valid.
const validKey = "valid"

type Params struct {
	fx.In

	Service       tenantports.AuthService
	PasswordReset *passwordresetservice.Service
	Logger        *zap.Logger
	Config        *config.Config
	ErrorHandler  *helpers.ErrorHandler
}

type Handler struct {
	service       tenantports.AuthService
	passwordReset *passwordresetservice.Service
	l             *zap.Logger
	cfg           *config.Config
	eh            *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service:       p.Service,
		passwordReset: p.PasswordReset,
		l:             p.Logger.With(zap.String("handler", "auth")),
		cfg:           p.Config,
		eh:            p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/auth/")
	api.POST("login", h.login)
	api.POST("mfa/verify", h.verifyMFA)
	api.POST("logout", h.logout)
	api.POST("validate-session", h.validateSession)
	api.POST("session/keepalive", h.keepSessionAlive)
	api.POST("forgot-password", h.forgotPassword)
	api.POST("reset-password", h.resetPassword)
	api.GET("session/roles", h.listAuthorizedSessionRoles)
	api.POST("session/roles/activate", h.activateSessionRoles)
	api.GET("csrf", h.csrfToken)
	api.GET("tenant/:slug", h.getTenantLoginMetadata)
	api.GET("providers/:orgSlug", h.listAuthProviders)
	api.GET("microsoft/start/:slug", h.startSSOLogin(tenant.SSOProviderAzureAD))
	api.GET("microsoft/callback", h.ssoCallback(tenant.SSOProviderAzureAD))
	api.GET("sso/start/:providerId/:orgSlug", h.startSSOLoginGeneric)
	api.GET("sso/callback/:provider", h.ssoCallbackGeneric)
}

func (h *Handler) listAuthorizedSessionRoles(c *gin.Context) {
	sess, ok := h.sessionFromCookie(c)
	if !ok {
		return
	}

	resp, err := h.service.ListAuthorizedSessionRoles(c.Request.Context(), sess.ID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) activateSessionRoles(c *gin.Context) {
	sess, ok := h.sessionFromCookie(c)
	if !ok {
		return
	}

	var req tenantports.ActivateSessionRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.SessionID = sess.ID

	resp, err := h.service.ActivateSessionRoles(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
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
	var req tenantports.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if resp.MFARequired {
		c.Header("Cache-Control", "no-store")
		helpers.JSON(c, http.StatusOK, resp)
		return
	}

	h.completeLogin(c, resp)
}

func (h *Handler) verifyMFA(c *gin.Context) {
	var req tenantports.VerifyMFAChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.VerifyMFAChallenge(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.completeLogin(c, resp)
}

func (h *Handler) completeLogin(c *gin.Context, resp *tenantports.LoginResponse) {
	authctx.SetAuthContext(
		c,
		resp.User.ID,
		resp.User.BusinessUnitID,
		resp.User.CurrentOrganizationID,
	)

	h.setSessionCookie(c, resp.SessionToken, resp.ExpiresAt)
	resp.CSRFToken = csrf.Token(resp.SessionID, h.cfg.Security.Session.Secret)

	helpers.JSON(c, http.StatusOK, resp)
}

// @Summary Logout
// @Description Invalidates the current session and clears the session cookie.
// @ID logout
// @Tags Auth
// @Success 204 "No Content"
// @Failure 401 {object} helpers.ProblemDetail
// @Router /auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	sess, ok := h.sessionFromCookie(c)
	if !ok {
		h.clearSessionCookie(c)
		return
	}

	if err := h.service.Logout(c.Request.Context(), sess.ID); err != nil {
		h.l.Warn("logout failed", zap.String("sessionID", sess.ID.String()), zap.Error(err))
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
	token, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || token == "" {
		helpers.JSON(c, http.StatusUnauthorized, gin.H{validKey: false})
		return
	}

	_, err = h.service.AuthenticateSession(c.Request.Context(), token)
	if err != nil {
		helpers.JSON(c, http.StatusUnauthorized, gin.H{validKey: false})
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{validKey: true})
}

// @Summary Keep session alive
// @Description Records that the person is still using the app, restarting the current session's idle timeout.
// @ID keepSessionAlive
// @Tags Auth
// @Success 204 "No Content"
// @Failure 401 {object} helpers.ProblemDetail
// @Router /auth/session/keepalive [post]
func (h *Handler) keepSessionAlive(c *gin.Context) {
	token, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || token == "" {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Session not found"))
		return
	}

	if _, err = h.service.KeepSessionAlive(c.Request.Context(), token); err != nil {
		h.clearSessionCookie(c)
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) csrfToken(c *gin.Context) {
	token, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || token == "" {
		h.anonymousCSRFToken(c)
		return
	}

	sess, err := h.service.AuthenticateSession(c.Request.Context(), token)
	if err != nil {
		h.anonymousCSRFToken(c)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{
		"csrfToken":  csrf.Token(sess.ID.String(), h.cfg.Security.Session.Secret),
		"headerName": h.cfg.Security.CSRF.HeaderName,
	})
}

func (h *Handler) anonymousCSRFToken(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, gin.H{
		"csrfToken":  "",
		"headerName": h.cfg.Security.CSRF.HeaderName,
	})
}

func (h *Handler) sessionFromCookie(c *gin.Context) (*session.Session, bool) {
	token, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || token == "" {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Session not found"))
		return nil, false
	}

	sess, err := h.service.AuthenticateSession(c.Request.Context(), token)
	if err != nil {
		h.eh.HandleError(c, err)
		return nil, false
	}

	return sess, true
}

func (h *Handler) getTenantLoginMetadata(c *gin.Context) {
	resp, err := h.service.GetTenantLoginMetadata(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) listAuthProviders(c *gin.Context) {
	resp, err := h.service.ListAuthProviders(c.Request.Context(), c.Param("orgSlug"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) startSSOLogin(provider tenant.SSOProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		redirectURL, err := h.service.StartSSOLogin(
			c.Request.Context(),
			tenantports.StartSSOLoginRequest{
				Provider:         provider,
				OrganizationSlug: c.Param("slug"),
				ReturnTo:         c.Query("returnTo"),
			},
		)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		c.Redirect(http.StatusFound, redirectURL)
	}
}

func (h *Handler) startSSOLoginGeneric(c *gin.Context) {
	rawProviderID := strings.TrimSpace(c.Param("providerId"))
	orgSlug := c.Param("orgSlug")
	if orgSlug == "" {
		orgSlug = c.Param("slug")
	}

	providerID, parseIDErr := pulid.Parse(rawProviderID)
	if parseIDErr == nil {
		redirectURL, err := h.service.StartSSOLogin(
			c.Request.Context(),
			tenantports.StartSSOLoginRequest{
				ProviderID:       providerID,
				OrganizationSlug: orgSlug,
				ReturnTo:         c.Query("returnTo"),
			},
		)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	provider, err := parseSSOProvider(rawProviderID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	redirectURL, err := h.service.StartSSOLogin(
		c.Request.Context(),
		tenantports.StartSSOLoginRequest{
			Provider:         provider,
			OrganizationSlug: orgSlug,
			ReturnTo:         c.Query("returnTo"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) ssoCallback(provider tenant.SSOProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := strings.TrimSpace(c.Query("state"))

		var loginState *tenantports.SSOLoginState
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

		resp, err := h.service.HandleSSOCallback(
			c.Request.Context(),
			tenantports.SSOCallbackRequest{
				State: state,
				Code:  strings.TrimSpace(c.Query("code")),
			},
		)
		if err != nil {
			h.redirectSSOError(c, loginState, err)
			return
		}

		if resp.LoginResponse.MFARequired {
			c.Redirect(http.StatusFound, h.ssoMFAChallengeURL(loginState, resp.LoginResponse))
			return
		}

		h.setSessionCookie(c, resp.LoginResponse.SessionToken, resp.LoginResponse.ExpiresAt)
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
		return "", errortypes.NewValidationError(
			"provider",
			errortypes.ErrInvalid,
			"Unsupported SSO provider",
		)
	}
}

func (h *Handler) redirectSSOError(
	c *gin.Context,
	loginState *tenantports.SSOLoginState,
	callbackErr error,
) {
	h.l.Warn("sso callback failed", zap.Error(callbackErr))

	loginPath := ssoLoginPath(loginState)

	userMsg := "Sign-in failed. Please try again or contact your administrator."

	origin := h.resolveSSOErrorOrigin(loginState)
	redirectURL := fmt.Sprintf("%s%s?sso_error=%s", origin, loginPath, url.QueryEscape(userMsg))
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *Handler) ssoMFAChallengeURL(
	loginState *tenantports.SSOLoginState,
	challenge *tenantports.LoginResponse,
) string {
	fragment := url.Values{}
	fragment.Set("mfa_challenge", challenge.MFAChallengeToken)
	fragment.Set("mfa_expires", strconv.FormatInt(challenge.ExpiresAt, 10))
	return h.resolveSSOErrorOrigin(loginState) + ssoLoginPath(loginState) + "#" + fragment.Encode()
}

func ssoLoginPath(loginState *tenantports.SSOLoginState) string {
	if loginState != nil && loginState.OrganizationSlug != "" {
		return "/login/" + url.PathEscape(loginState.OrganizationSlug)
	}
	return "/login"
}

func (h *Handler) resolveSSOErrorOrigin(loginState *tenantports.SSOLoginState) string {
	if loginState != nil {
		if returnURL, err := url.Parse(loginState.ReturnTo); err == nil &&
			(returnURL.Scheme == "http" || returnURL.Scheme == "https") &&
			returnURL.Host != "" {
			return returnURL.Scheme + "://" + returnURL.Host
		}
	}

	if len(h.cfg.Server.CORS.AllowedOrigins) > 0 {
		return h.cfg.Server.CORS.AllowedOrigins[0]
	}

	return ""
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	sessioncookie.Clear(c, &h.cfg.Security.Session)
}

func (h *Handler) setSessionCookie(c *gin.Context, sessionID string, expiresAt int64) {
	sessioncookie.Set(c, &h.cfg.Security.Session, sessionID, expiresAt)
}
