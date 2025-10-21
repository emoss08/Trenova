package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type APITokenHandlerParams struct {
	fx.In

	AuthService  services.AuthService
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
}

type APITokenHandler struct {
	authService services.AuthService
	l           *zap.Logger
	eh          *helpers.ErrorHandler
}

func NewAPITokenHandler(p APITokenHandlerParams) *APITokenHandler {
	return &APITokenHandler{
		authService: p.AuthService,
		l:           p.Logger.Named("apitoken-handler"),
		eh:          p.ErrorHandler,
	}
}

func (h *APITokenHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tokens := rg.Group("/api-tokens/")
	tokens.GET("", h.listTokens)
	tokens.POST("", h.createToken)
	tokens.DELETE(":id/", h.revokeToken)
}

type CreateTokenRequest struct {
	Name        string                 `json:"name"        binding:"required,min=1,max=100"`
	Description string                 `json:"description" binding:"max=500"`
	Scopes      []tenant.APITokenScope `json:"scopes"      binding:"required,min=1"`
	ExpiresAt   *int64                 `json:"expires_at"`
}

type CreateTokenResponse struct {
	Token      *tenant.APIToken `json:"token"`
	PlainToken string           `json:"plain_token"`
}

func (h *APITokenHandler) listTokens(c *gin.Context) {
	userID, exists := context.GetUserID(c)
	if !exists {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("User not authenticated"))
		return
	}

	tokens, err := h.authService.ListUserAPITokens(c.Request.Context(), userID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

func (h *APITokenHandler) createToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()),
		)
		return
	}

	userID, exists := context.GetUserID(c)
	if !exists {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("User not authenticated"))
		return
	}

	buID, exists := context.GetBusinessUnitID(c)
	if !exists {
		h.eh.HandleError(c, errortypes.NewBusinessError("Business unit not found in context"))
		return
	}

	orgID, exists := context.GetOrganizationID(c)
	if !exists {
		h.eh.HandleError(c, errortypes.NewBusinessError("Organization not found in context"))
		return
	}

	resp, err := h.authService.CreateAPIToken(c.Request.Context(), &services.CreateAPITokenRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Scopes:         req.Scopes,
		ExpiresAt:      req.ExpiresAt,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp.Token.SanitizeForResponse()

	c.JSON(http.StatusCreated, CreateTokenResponse{
		Token:      resp.Token,
		PlainToken: resp.PlainToken,
	})
}

func (h *APITokenHandler) revokeToken(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := pulid.Parse(tokenIDStr)
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"tokenId",
				errortypes.ErrInvalidFormat,
				"Invalid token ID",
			),
		)
		return
	}

	userID, exists := context.GetUserID(c)
	if !exists {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("User not authenticated"))
		return
	}

	tokens, err := h.authService.ListUserAPITokens(c.Request.Context(), userID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tokenBelongsToUser := false
	for _, token := range tokens {
		if token.ID == tokenID {
			tokenBelongsToUser = true
			break
		}
	}

	if !tokenBelongsToUser {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("Token not found or access denied"))
		return
	}

	if err = h.authService.RevokeAPIToken(c.Request.Context(), services.RevokeAPITokenRequest{
		TokenID: tokenID,
		UserID:  userID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token revoked successfully",
		"id":      tokenID.String(),
	})
}
