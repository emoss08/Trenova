package handlers

import (
	"net/http"
	"time"

	authcontext "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AuthHandlerParams struct {
	fx.In

	AuthService  services.AuthService
	Config       *config.Config
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
}

type AuthHandler struct {
	authService services.AuthService
	cfg         *config.Config
	l           *zap.Logger
	eh          *helpers.ErrorHandler
}

func NewAuthHandler(p AuthHandlerParams) *AuthHandler {
	return &AuthHandler{
		authService: p.AuthService,
		cfg:         p.Config,
		l:           p.Logger.With(zap.String("handler", "auth")),
		eh:          p.ErrorHandler,
	}
}

func (h *AuthHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login/", h.login)
	auth.POST("/logout/", h.logout)
	auth.POST("/check-email/", h.checkEmail)
	auth.POST("/validate-session/", h.validateSession)
}

type LoginRequest struct {
	EmailAddress string `json:"emailAddress" binding:"required,email"`
	Password     string `json:"password"     binding:"required"`
}

func (h *AuthHandler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	loginReq := services.LoginRequest{
		EmailAddress: req.EmailAddress,
		Password:     req.Password,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	}

	resp, err := h.authService.Login(c.Request.Context(), loginReq)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.setSessionCookie(c, resp.SessionID, resp.ExpiresAt)

	authcontext.SetUserID(c, resp.User.ID)
	authcontext.SetBusinessUnitID(c, resp.User.BusinessUnitID)
	authcontext.SetOrganizationID(c, resp.User.CurrentOrganizationID)
	authcontext.SetAuthType(c, authcontext.AuthTypeSession)

	h.l.Debug("User logged in successfully",
		zap.String("userId", resp.User.ID.String()),
		zap.String("email", resp.User.EmailAddress),
		zap.String("ip", c.ClientIP()),
	)

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) logout(c *gin.Context) {
	sessionIDStr, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || sessionIDStr == "" {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("No session found"))
		return
	}

	sessionID, err := pulid.Parse(sessionIDStr)
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"sessionId",
				errortypes.ErrInvalidFormat,
				"Invalid session ID",
			),
		)
		return
	}

	logoutReq := services.LogoutRequest{
		SessionID: sessionID,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Reason:    "User initiated logout",
	}

	if err = h.authService.Logout(c.Request.Context(), logoutReq); err != nil {
		h.l.Warn("Logout failed",
			zap.String("sessionId", sessionID.String()),
			zap.Error(err),
		)
	}

	h.clearSessionCookie(c)

	h.l.Info("User logged out successfully",
		zap.String("sessionId", sessionID.String()),
		zap.String("ip", c.ClientIP()),
	)

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) checkEmail(c *gin.Context) {
	var req services.CheckEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	if err := req.Validate(); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	exists, err := h.authService.CheckEmail(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": exists})
}

func (h *AuthHandler) validateSession(c *gin.Context) {
	sessionIDStr, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || sessionIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
		return
	}

	sessionID, err := pulid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false})
		return
	}

	valid, err := h.authService.ValidateSession(
		c.Request.Context(),
		services.ValidateSessionRequest{
			SessionID: sessionID,
			ClientIP:  c.ClientIP(),
		},
	)
	if err != nil {
		h.l.Debug("Session validation failed",
			zap.String("sessionId", sessionID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, sessionID string, expiresAt int64) {
	maxAge := max(0, int(expiresAt-time.Now().Unix()))

	c.SetCookie(
		h.cfg.Security.Session.Name,
		sessionID,
		maxAge,
		h.cfg.Security.Session.Path,
		h.cfg.Security.Session.Domain,
		h.cfg.Security.Session.Secure,
		h.cfg.Security.Session.HTTPOnly,
	)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetCookie(
		h.cfg.Security.Session.Name,
		"",
		-1,
		h.cfg.Security.Session.Path,
		h.cfg.Security.Session.Domain,
		h.cfg.Security.Session.Secure,
		h.cfg.Security.Session.HTTPOnly,
	)
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}
