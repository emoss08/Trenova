// Package inboundhandler receives forwarded mail from a provider.
//
// The endpoint is unauthenticated because a mail provider cannot hold a
// session. Everything about who a delivery belongs to comes from the token in
// its URL, and everything about whether to believe it comes from its signature
// — both checked in the service, so the handler cannot be the place a check is
// accidentally skipped.
package inboundhandler

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// maxInboundBodyBytes is sized for mail, not for an event hook.
//
// A forwarded tender carries its PDF base64-encoded inside the body, so the
// 1 MiB the outbound delivery hooks use would reject the ordinary case. This is
// the AS2 endpoint's limit, which was chosen for the same reason.
const maxInboundBodyBytes = 32 << 20

type Params struct {
	fx.In

	Service      *inboundmessageservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *inboundmessageservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler}
}

// RegisterPublicRoutes mounts the ingest endpoint.
//
// The path is deliberately its own prefix rather than hanging off
// /webhooks/email/, where the outbound delivery hooks already own the segment
// after the provider name with a wildcard.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/inbound-mail/:mailboxToken/", h.receive)
}

func (h *Handler) receive(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxInboundBodyBytes))
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	result, err := h.service.ReceiveWebhook(c.Request.Context(), &inboundmessageservice.ReceiveWebhookRequest{
		MailboxToken:       c.Param("mailboxToken"),
		Body:               body,
		SignatureID:        c.GetHeader("svix-id"),
		SignatureTimestamp: c.GetHeader("svix-timestamp"),
		Signature:          c.GetHeader("svix-signature"),
		Authorization:      c.GetHeader("Authorization"),
		ReceivedAt:         time.Now(),
	})
	if err != nil {
		h.reject(c, err)

		return
	}

	// A duplicate and a first delivery are both a success as far as the
	// provider is concerned. Reporting anything else would make it retry a
	// message that is already handled.
	c.JSON(http.StatusOK, gin.H{
		"messageId": result.MessageID,
		"duplicate": result.Duplicate,
		"ignored":   result.Ignored,
	})
}

// reject answers without saying which check failed.
//
// An unknown token and an inactive mailbox both get a flat 404: an endpoint
// that distinguished them would let anyone discover which addresses exist by
// posting to guesses. A delivery that verified but was malformed gets 400,
// because that one is genuinely the provider's problem and it should not retry
// it forever.
func (h *Handler) reject(c *gin.Context, err error) {
	switch {
	case errors.Is(err, inboundmessageservice.ErrUnknownMailbox),
		errors.Is(err, inboundmessageservice.ErrMailboxInactive):
		c.Status(http.StatusNotFound)
	case errors.Is(err, inboundmessageservice.ErrUnverified):
		c.Status(http.StatusUnauthorized)
	case errors.Is(err, inboundmessageservice.ErrMalformedPayload),
		errors.Is(err, inboundmessageservice.ErrUnsupportedProvider):
		c.Status(http.StatusBadRequest)
	default:
		h.eh.HandleError(c, err)
	}
}
