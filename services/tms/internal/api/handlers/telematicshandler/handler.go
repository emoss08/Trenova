package telematicshandler

import (
	"io"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/samsara/webhooks"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const maxWebhookBodyBytes = 1 << 20

type Params struct {
	fx.In

	Service      *telematicsservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *telematicsservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

var webhookProviders = map[string]integration.Type{
	"samsara": integration.TypeSamsara,
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/samsara/:webhookToken/", h.handleProviderWebhook(integration.TypeSamsara))
	rg.POST("/webhooks/telematics/:provider/:webhookToken/", h.handleTelematicsWebhook)
}

func (h *Handler) handleTelematicsWebhook(c *gin.Context) {
	providerType, ok := webhookProviders[strings.ToLower(c.Param("provider"))]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	h.processWebhook(c, providerType)
}

func (h *Handler) handleProviderWebhook(providerType integration.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.processWebhook(c, providerType)
	}
}

func (h *Handler) processWebhook(c *gin.Context, providerType integration.Type) {
	token := c.Param("webhookToken")
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.ProcessWebhook(c.Request.Context(), &telematicsservice.ProcessWebhookRequest{
		ProviderType: providerType,
		Token:        token,
		Body:         body,
		Signature:    c.GetHeader(webhooks.HeaderSignature),
		Timestamp:    c.GetHeader(webhooks.HeaderTimestamp),
	})
	if err != nil {
		if errortypes.IsError(err) {
			c.Status(http.StatusUnauthorized)
			return
		}
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
