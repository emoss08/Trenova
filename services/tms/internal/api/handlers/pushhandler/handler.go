package pushhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/webpushservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service      *webpushservice.Service
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *webpushservice.Service
	l       *zap.Logger
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		l:       p.Logger.With(zap.String("handler", "push")),
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/push/")
	api.GET("public-key/", h.publicKey)
	api.POST("subscriptions/", h.subscribe)
	api.DELETE("subscriptions/", h.unsubscribe)
}

type publicKeyResponse struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"publicKey"`
}

// @Summary Get the VAPID public key for web push
// @ID getPushPublicKey
// @Tags Push
// @Produce json
// @Success 200 {object} publicKeyResponse
// @Router /push/public-key [get]
func (h *Handler) publicKey(c *gin.Context) {
	c.JSON(http.StatusOK, publicKeyResponse{
		Enabled:   h.service.Enabled(),
		PublicKey: h.service.PublicKey(),
	})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh"   binding:"required"`
	Auth     string `json:"auth"     binding:"required"`
}

// @Summary Register a web push subscription for the authenticated user
// @ID createPushSubscription
// @Tags Push
// @Accept json
// @Produce json
// @Param request body subscribeRequest true "Push subscription keys"
// @Success 201 {object} notification.PushSubscription
// @Failure 422 {object} helpers.ProblemDetail
// @Router /push/subscriptions [post]
func (h *Handler) subscribe(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	subscription, err := h.service.Subscribe(
		c.Request.Context(),
		&repositories.SavePushSubscriptionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Endpoint:  req.Endpoint,
			P256dh:    req.P256dh,
			Auth:      req.Auth,
			UserAgent: c.Request.UserAgent(),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, subscription)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
}

// @Summary Remove a web push subscription for the authenticated user
// @ID deletePushSubscription
// @Tags Push
// @Accept json
// @Success 204 "No Content"
// @Failure 422 {object} helpers.ProblemDetail
// @Router /push/subscriptions [delete]
func (h *Handler) unsubscribe(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req unsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := h.service.Unsubscribe(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		req.Endpoint,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
