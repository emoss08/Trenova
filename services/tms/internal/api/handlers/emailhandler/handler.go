package emailhandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *emailservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *emailservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	profiles := rg.Group("/email-profiles")
	profiles.GET("/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead), h.listProfiles)
	profiles.POST("/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpCreate), h.createProfile)
	profiles.GET("/select-options/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead), h.selectProfileOptions)
	profiles.GET("/select-options/:profileID", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead), h.getProfileOption)
	profiles.GET("/:profileID/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead), h.getProfile)
	profiles.PUT("/:profileID/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate), h.updateProfile)
	profiles.DELETE("/:profileID/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpDelete), h.deleteProfile)
	profiles.POST("/:profileID/test-send/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate), h.testSend)
	profiles.GET("/assignments/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead), h.listAssignments)
	profiles.PUT("/assignments/", h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate), h.updateAssignments)

	logs := rg.Group("/email-logs")
	logs.GET("/", h.pm.RequirePermission(permission.ResourceEmailLog.String(), permission.OpRead), h.listLogs)
	logs.GET("/:messageID/", h.pm.RequirePermission(permission.ResourceEmailLog.String(), permission.OpRead), h.getLog)

	suppressions := rg.Group("/email-suppressions")
	suppressions.GET("/", h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpRead), h.listSuppressions)
	suppressions.POST("/", h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpCreate), h.createSuppression)
	suppressions.DELETE("/:suppressionID/", h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpDelete), h.deleteSuppression)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/email/resend/:webhookToken/", h.handleResendWebhook)
	rg.POST("/webhooks/email/postmark/:webhookToken/", h.handlePostmarkWebhook)
}

func (h *Handler) listProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Profile], error) {
		return h.service.ListProfiles(c.Request.Context(), &repositories.ListEmailProfilesRequest{
			Filter: req,
		})
	})
}

func (h *Handler) selectProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*email.Profile], error) {
			return h.service.SelectProfileOptions(
				c.Request.Context(),
				&repositories.EmailProfileSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

func (h *Handler) getProfileOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile, err := h.service.GetProfile(c.Request.Context(), repositories.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) getProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile, err := h.service.GetProfile(c.Request.Context(), repositories.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) createProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profile := new(email.Profile)
	authctx.AddContextToRequest(authCtx, profile)
	if err := c.ShouldBindJSON(profile); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile.OrganizationID = authCtx.OrganizationID
	profile.BusinessUnitID = authCtx.BusinessUnitID
	created, err := h.service.CreateProfile(c.Request.Context(), profile, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) updateProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile := new(email.Profile)
	authctx.AddContextToRequest(authCtx, profile)
	if err = c.ShouldBindJSON(profile); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile.ID = id
	profile.OrganizationID = authCtx.OrganizationID
	profile.BusinessUnitID = authCtx.BusinessUnitID
	updated, err := h.service.UpdateProfile(c.Request.Context(), profile, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.DeleteProfile(c.Request.Context(), repositories.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) testSend(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(services.TestEmailProfileRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	msg, err := h.service.TestSend(c.Request.Context(), tenantInfo(authCtx), id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, msg)
}

func (h *Handler) listAssignments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	assignments, err := h.service.ListAssignments(c.Request.Context(), tenantInfo(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, assignments)
}

func (h *Handler) updateAssignments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	assignments := make([]*email.ProfileAssignment, 0, len(email.Purposes()))
	if err := c.ShouldBindJSON(&assignments); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.UpsertAssignments(
		c.Request.Context(),
		tenantInfo(authCtx),
		assignments,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) listLogs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Message], error) {
		return h.service.ListMessages(c.Request.Context(), &repositories.ListEmailMessagesRequest{
			Filter: req,
		})
	})
}

func (h *Handler) getLog(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	msg, err := h.service.GetMessage(c.Request.Context(), repositories.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, msg)
}

func (h *Handler) listSuppressions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Suppression], error) {
		return h.service.ListSuppressions(c.Request.Context(), &repositories.ListEmailSuppressionsRequest{
			Filter: req,
		})
	})
}

func (h *Handler) createSuppression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	suppression := new(email.Suppression)
	if err := c.ShouldBindJSON(suppression); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	suppression.OrganizationID = authCtx.OrganizationID
	suppression.BusinessUnitID = authCtx.BusinessUnitID
	suppression.CreatedByID = authCtx.UserID
	created, err := h.service.CreateSuppression(c.Request.Context(), suppression)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) deleteSuppression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("suppressionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.DeleteSuppression(c.Request.Context(), repositories.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type resendWebhookPayload struct {
	Type string `json:"type"`
	Data struct {
		ID      string `json:"id"`
		EmailID string `json:"email_id"`
		To      string `json:"to"`
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
	tenantInfo, signingSecret, err := h.service.ResolveTenantByWebhookToken(c.Request.Context(), token)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = verifySvixSignature(svixSignatureParams{
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
	providerEventID := payload.Data.ID
	if providerEventID == "" {
		providerEventID = c.GetHeader("svix-id")
	}
	event := &email.Event{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        email.ProviderResend,
		ProviderEventID: providerEventID,
		Type:            resendEventType(payload.Type),
		Recipient:       payload.Data.To,
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

type svixSignatureParams struct {
	Secret    string
	ID        string
	Timestamp string
	Signature string
	Body      []byte
}

func verifySvixSignature(p svixSignatureParams) error {
	if p.Secret == "" || p.ID == "" || p.Timestamp == "" || p.Signature == "" {
		return errors.New("missing svix signature headers")
	}
	secret := strings.TrimPrefix(p.Secret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(p.ID + "." + p.Timestamp + "."))
	mac.Write(p.Body)
	expected := mac.Sum(nil)
	for _, part := range strings.Split(p.Signature, " ") {
		_, sig, ok := strings.Cut(part, ",")
		if !ok {
			sig = strings.TrimPrefix(part, "v1,")
		}
		decoded, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return errors.New("invalid svix signature")
}

func resendEventType(value string) email.EventType {
	switch value {
	case "email.delivered":
		return email.EventTypeDelivered
	case "email.opened":
		return email.EventTypeOpened
	case "email.clicked":
		return email.EventTypeClicked
	case "email.bounced":
		return email.EventTypeBounced
	case "email.complained":
		return email.EventTypeComplained
	default:
		return email.EventTypeFailed
	}
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
	event := &email.Event{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        email.ProviderPostmark,
		ProviderEventID: postmarkProviderEventID(payload),
		Type:            postmarkEventType(payload.RecordType),
		Recipient:       postmarkRecipient(payload),
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
		SuppressionReason: postmarkSuppressionReason(payload),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func postmarkEventType(value string) email.EventType {
	switch value {
	case "Delivery":
		return email.EventTypeDelivered
	case "Open":
		return email.EventTypeOpened
	case "Click":
		return email.EventTypeClicked
	case "Bounce":
		return email.EventTypeBounced
	case "SpamComplaint":
		return email.EventTypeComplained
	default:
		return email.EventTypeFailed
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

func postmarkSuppressionReason(payload postmarkWebhookPayload) email.SuppressionReason {
	switch {
	case payload.RecordType == "Bounce" && payload.Type == "HardBounce":
		return email.SuppressionReasonHardBounce
	case payload.RecordType == "SpamComplaint":
		return email.SuppressionReasonComplaint
	default:
		return ""
	}
}

func postmarkProviderEventID(payload postmarkWebhookPayload) string {
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

func postmarkRecipient(payload postmarkWebhookPayload) string {
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

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
