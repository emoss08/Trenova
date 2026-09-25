package accountingwebhookhandler

import (
	"io"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const maxWebhookBodyBytes = 1 << 20

var providers = map[string]integration.Type{
	"quickbooks": integration.TypeQuickBooksOnline,
}

type Params struct {
	fx.In

	Service      services.AccountingConnectionService
	Connectors   services.AccountingConnectorRegistry
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service    services.AccountingConnectionService
	connectors services.AccountingConnectorRegistry
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, connectors: p.Connectors, eh: p.ErrorHandler}
}

func WebhookPath(provider string) string {
	return "/webhooks/accounting/" + provider + "/"
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/accounting/:provider/", h.receive)
}

func (h *Handler) receive(c *gin.Context) {
	typ, ok := providers[strings.ToLower(c.Param("provider"))]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	provider, ok := h.connectors.For(typ)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes+1))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if len(body) > maxWebhookBodyBytes {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}

	err = h.service.ReceiveWebhook(c.Request.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: typ,
		Signature:       c.GetHeader(provider.WebhookSignatureHeader()),
		Body:            body,
	})
	switch {
	case err == nil:
		c.Status(http.StatusOK)
	case errortypes.IsAuthenticationError(err):
		c.Status(http.StatusUnauthorized)
	case errortypes.IsError(err):
		c.Status(http.StatusBadRequest)
	default:
		h.eh.HandleError(c, err)
	}
}
