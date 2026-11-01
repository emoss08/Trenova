package emailhandler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *emailservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *emailservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/email/resend/:webhookToken/", h.handleResendWebhook)
	rg.POST("/webhooks/email/postmark/:webhookToken/", h.handlePostmarkWebhook)
}

type resendWebhookPayload struct {
	Type string `json:"type"`
	Data struct {
		ID      string              `json:"id"`
		EmailID string              `json:"email_id"`
		To      jsonflex.StringList `json:"to"`
		Bounce  struct {
			Type string `json:"type"`
		} `json:"bounce"`
	} `json:"data"`
	CreatedAt string `json:"created_at"`
}

type postmarkWebhookPayload struct {
	RecordType  string `json:"RecordType"`
	ID          any    `json:"ID"`
	MessageID   string `json:"MessageID"`
	Recipient   string `json:"Recipient"`
	Email       string `json:"Email"`
	Type        string `json:"Type"`
	ReceivedAt  string `json:"ReceivedAt"`
	DeliveredAt string `json:"DeliveredAt"`
	BouncedAt   string `json:"BouncedAt"`
}

func (h *Handler) handleResendWebhook(c *gin.Context) {
	token := c.Param("webhookToken")
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo, signingSecret, err := h.service.ResolveTenantByWebhookToken(
		c.Request.Context(),
		token,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = webhooksig.VerifySvix(webhooksig.SvixParams{
		Secret:    signingSecret,
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
	eventType, known := resendEventType(&payload)
	if !known {
		c.Status(http.StatusNoContent)
		return
	}
	providerEventID := c.GetHeader("svix-id")
	if providerEventID == "" {
		providerEventID = payload.Data.ID
	}
	event := &email.Event{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        email.ProviderResend,
		ProviderEventID: providerEventID,
		Type:            eventType,
		Recipient:       resendRecipient(&payload),
		OccurredAt:      timeutils.NowUnix(),
		Raw:             map[string]any{},
	}
	if err = sonic.Unmarshal(body, &event.Raw); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.HandleProviderEvent(c.Request.Context(), emailservice.HandleProviderEventParams{
		TenantInfo:        tenantInfo,
		Event:             event,
		ProviderMessageID: payload.Data.EmailID,
		SuppressionReason: resendSuppressionReason(event.Type),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func resendEventType(payload *resendWebhookPayload) (email.EventType, bool) {
	switch payload.Type {
	case "email.sent":
		return email.EventTypeSent, true
	case "email.delivered":
		return email.EventTypeDelivered, true
	case "email.opened":
		return email.EventTypeOpened, true
	case "email.clicked":
		return email.EventTypeClicked, true
	case "email.bounced":
		if strings.EqualFold(payload.Data.Bounce.Type, "Transient") {
			return "", false
		}
		return email.EventTypeBounced, true
	case "email.complained":
		return email.EventTypeComplained, true
	case "email.failed":
		return email.EventTypeFailed, true
	default:
		return "", false
	}
}

func resendRecipient(payload *resendWebhookPayload) string {
	if len(payload.Data.To) != 1 {
		return ""
	}
	return payload.Data.To.First()
}

func (h *Handler) handlePostmarkWebhook(c *gin.Context) {
	token := c.Param("webhookToken")
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo, err := h.service.ResolveTenantByProviderWebhookToken(
		c.Request.Context(),
		email.ProviderPostmark,
		token,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var payload postmarkWebhookPayload
	if err = sonic.Unmarshal(body, &payload); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	eventType, known := postmarkEventType(&payload)
	if !known {
		c.Status(http.StatusNoContent)
		return
	}
	event := &email.Event{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        email.ProviderPostmark,
		ProviderEventID: postmarkProviderEventID(&payload),
		Type:            eventType,
		Recipient:       postmarkRecipient(&payload),
		OccurredAt:      timeutils.NowUnix(),
		Raw:             map[string]any{},
	}
	if err = sonic.Unmarshal(body, &event.Raw); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.HandleProviderEvent(c.Request.Context(), emailservice.HandleProviderEventParams{
		TenantInfo:        tenantInfo,
		Event:             event,
		ProviderMessageID: payload.MessageID,
		SuppressionReason: postmarkSuppressionReason(&payload),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

var postmarkHardBounceTypes = map[string]struct{}{
	"HardBounce":          {},
	"BadEmailAddress":     {},
	"ManuallyDeactivated": {},
	"SpamNotification":    {},
}

func postmarkIsHardBounce(payload *postmarkWebhookPayload) bool {
	_, ok := postmarkHardBounceTypes[payload.Type]
	return ok
}

func postmarkEventType(payload *postmarkWebhookPayload) (email.EventType, bool) {
	switch payload.RecordType {
	case "Delivery":
		return email.EventTypeDelivered, true
	case "Open":
		return email.EventTypeOpened, true
	case "Click":
		return email.EventTypeClicked, true
	case "Bounce":
		if !postmarkIsHardBounce(payload) {
			return "", false
		}
		return email.EventTypeBounced, true
	case "SpamComplaint":
		return email.EventTypeComplained, true
	default:
		return "", false
	}
}

func resendSuppressionReason(eventType email.EventType) email.SuppressionReason {
	switch eventType {
	case email.EventTypeBounced:
		return email.SuppressionReasonHardBounce
	case email.EventTypeComplained:
		return email.SuppressionReasonComplaint
	default:
		return ""
	}
}

func postmarkSuppressionReason(payload *postmarkWebhookPayload) email.SuppressionReason {
	switch {
	case payload.RecordType == "Bounce" && postmarkIsHardBounce(payload):
		return email.SuppressionReasonHardBounce
	case payload.RecordType == "SpamComplaint":
		return email.SuppressionReasonComplaint
	default:
		return ""
	}
}

func postmarkProviderEventID(payload *postmarkWebhookPayload) string {
	if id := postmarkStringValue(payload.ID); id != "" {
		return id
	}
	timestamp := payload.DeliveredAt
	if timestamp == "" {
		timestamp = payload.BouncedAt
	}
	if timestamp == "" {
		timestamp = payload.ReceivedAt
	}
	return strings.Join(
		[]string{payload.RecordType, payload.MessageID, postmarkRecipient(payload), timestamp},
		":",
	)
}

func postmarkRecipient(payload *postmarkWebhookPayload) string {
	if payload.Recipient != "" {
		return payload.Recipient
	}
	return payload.Email
}

func postmarkStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}
