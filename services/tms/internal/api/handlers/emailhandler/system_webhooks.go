package emailhandler

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/gin-gonic/gin"
)

const systemWebhookPath = "/webhooks/email/system/"

func (h *Handler) handleSystemResendWebhook(c *gin.Context) {
	body, err := helpers.ReadBoundedRequestBody(c, maxWebhookBodyBytes)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	secret, ok := h.systemWebhookSecret(c, email.ProviderResend)
	if !ok {
		return
	}
	if err = webhooksig.VerifySvix(webhooksig.SvixParams{
		Secret:    secret,
		ID:        c.GetHeader("svix-id"),
		Timestamp: c.GetHeader("svix-timestamp"),
		Signature: c.GetHeader("svix-signature"),
		Body:      body,
	}); err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}
	var payload resendWebhookPayload
	if err = sonic.Unmarshal(body, &payload); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenant, found, err := h.service.TenantOfSystemMessage(
		c.Request.Context(),
		email.ProviderResend,
		payload.Data.EmailID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !found {
		c.Status(http.StatusNoContent)
		return
	}
	h.recordResendEvent(c, tenant, body, &payload)
}

func (h *Handler) handleSystemPostmarkWebhook(c *gin.Context) {
	body, err := helpers.ReadBoundedRequestBody(c, maxWebhookBodyBytes)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	credentials, ok := h.systemWebhookSecret(c, email.ProviderPostmark)
	if !ok {
		return
	}
	if err = webhooksig.VerifyBasicAuth(credentials, c.GetHeader("Authorization")); err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}
	var payload postmarkWebhookPayload
	if err = sonic.Unmarshal(body, &payload); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenant, found, err := h.service.TenantOfSystemMessage(
		c.Request.Context(),
		email.ProviderPostmark,
		payload.MessageID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !found {
		c.Status(http.StatusNoContent)
		return
	}
	h.recordPostmarkEvent(c, tenant, body, &payload)
}

func (h *Handler) systemWebhookSecret(c *gin.Context, provider email.Provider) (string, bool) {
	secret, ok, err := h.service.SystemWebhookSecret(c.Request.Context(), provider)
	if err != nil {
		h.eh.HandleError(c, err)
		return "", false
	}
	if !ok {
		c.Status(http.StatusNotFound)
		return "", false
	}
	return secret, true
}
