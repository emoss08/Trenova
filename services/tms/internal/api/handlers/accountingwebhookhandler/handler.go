package accountingwebhookhandler

import (
	"io"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const maxWebhookBodyBytes = 1 << 20

type Params struct {
	fx.In

	Webhooks     accountingconnectionports.AccountingWebhookReceiver
	Connectors   accountingconnectorports.AccountingConnectorRegistry
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	webhooks   accountingconnectionports.AccountingWebhookReceiver
	connectors accountingconnectorports.AccountingConnectorRegistry
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{webhooks: p.Webhooks, connectors: p.Connectors, eh: p.ErrorHandler}
}

func WebhookPath(provider string) string {
	return "/webhooks/accounting/" + provider + "/"
}

const validationTokenParam = "validationToken"

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/accounting/:provider/", h.receive)
	rg.POST("/webhooks/accounting/:provider/:app/", h.receive)
	rg.GET("/webhooks/accounting/:provider/", h.handshake)
	rg.GET("/webhooks/accounting/:provider/:app/", h.handshake)
}

func (h *Handler) resolve(
	c *gin.Context,
) (*accountingprovider.ProviderProfile, accountingconnectorports.AccountingProvider, bool) {
	profile, ok := accountingprovider.ProfileByWebhookSlug(strings.ToLower(c.Param("provider")))
	if !ok {
		return nil, nil, false
	}
	provider, ok := h.connectors.For(profile.Type)
	if !ok {
		return nil, nil, false
	}
	return &profile, provider, true
}

func (h *Handler) handshake(c *gin.Context) {
	profile, provider, ok := h.resolve(c)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	if !h.answerHandshake(c, profile, provider) {
		c.Status(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) answerHandshake(
	c *gin.Context,
	profile *accountingprovider.ProviderProfile,
	provider accountingconnectorports.AccountingProvider,
) bool {
	raw, present := c.GetQuery(validationTokenParam)
	if !present || !profile.WebhookSubscriptions {
		return false
	}
	subscribing, ok := provider.(accountingconnectorports.AccountingSubscriptionProvider)
	if !ok {
		return false
	}
	token, valid := subscribing.ValidationToken(raw)
	if !valid {
		c.Status(http.StatusBadRequest)
		return true
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(token))
	return true
}

func (h *Handler) receive(c *gin.Context) {
	profile, provider, ok := h.resolve(c)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	if h.answerHandshake(c, profile, provider) {
		return
	}
	typ := profile.Type

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes+1))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if len(body) > maxWebhookBodyBytes {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}

	err = h.webhooks.ReceiveWebhook(
		c.Request.Context(),
		&accountingconnectionports.ReceiveAccountingWebhookRequest{
			IntegrationType: typ,
			AppID:           c.Param("app"),
			Signature:       c.GetHeader(provider.WebhookSignatureHeader()),
			Body:            body,
		},
	)
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
