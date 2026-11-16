package emailhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const maxWebhookBodyBytes = 1 << 20

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
	profiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead),
		h.listProfiles,
	)
	profiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpCreate),
		h.createProfile,
	)
	profiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead),
		h.getProfile,
	)
	profiles.PUT(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate),
		h.updateProfile,
	)
	profiles.DELETE(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpDelete),
		h.deleteProfile,
	)
	profiles.POST(
		"/:profileID/test-send/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate),
		h.testSend,
	)
	profiles.GET(
		"/assignments/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpRead),
		h.listAssignments,
	)
	profiles.PUT(
		"/assignments/",
		h.pm.RequirePermission(permission.ResourceEmailProfile.String(), permission.OpUpdate),
		h.updateAssignments,
	)

	logs := rg.Group("/email-logs")
	logs.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEmailLog.String(), permission.OpRead),
		h.listLogs,
	)
	logs.GET(
		"/:messageID/",
		h.pm.RequirePermission(permission.ResourceEmailLog.String(), permission.OpRead),
		h.getLog,
	)

	suppressions := rg.Group("/email-suppressions")
	suppressions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpRead),
		h.listSuppressions,
	)
	suppressions.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpCreate),
		h.createSuppression,
	)
	suppressions.DELETE(
		"/:suppressionID/",
		h.pm.RequirePermission(permission.ResourceEmailSuppression.String(), permission.OpDelete),
		h.deleteSuppression,
	)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/email/resend/:webhookToken/", h.handleResendWebhook)
	rg.POST("/webhooks/email/postmark/:webhookToken/", h.handlePostmarkWebhook)
}

func (h *Handler) listProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Profile], error) {
		return h.service.ListProfiles(c.Request.Context(), &emailports.ListEmailProfilesRequest{
			Filter: req,
		})
	})
}

func (h *Handler) getProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile, err := h.service.GetProfile(c.Request.Context(), emailports.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, profile)
}

func (h *Handler) createProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profile := new(email.Profile)
	if err := authctx.BindJSON(c, authCtx, profile); err != nil {
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
	helpers.JSON(c, http.StatusCreated, created)
}

func (h *Handler) updateProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	profile := new(email.Profile)
	if err = authctx.BindJSON(c, authCtx, profile); err != nil {
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
	helpers.JSON(c, http.StatusOK, updated)
}

func (h *Handler) deleteProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.DeleteProfile(c.Request.Context(), emailports.GetEmailEntityRequest{
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
	req := new(emailports.TestEmailProfileRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	msg, err := h.service.TestSend(c.Request.Context(), tenantInfo(authCtx), id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusAccepted, msg)
}

func (h *Handler) listAssignments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	tenant := tenantInfo(authCtx)
	assignments, err := h.service.ListAssignments(c.Request.Context(), tenant)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	routing, err := h.service.Routing(c.Request.Context(), tenant, assignments)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, routing)
}

func (h *Handler) updateAssignments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	assignments := make([]*email.ProfileAssignment, 0, len(email.Purposes()))
	if err := c.ShouldBindJSON(&assignments); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenant := tenantInfo(authCtx)
	updated, err := h.service.UpsertAssignments(
		c.Request.Context(),
		tenant,
		assignments,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	routing, err := h.service.Routing(c.Request.Context(), tenant, updated)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, routing)
}

func (h *Handler) listLogs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Message], error) {
		return h.service.ListMessages(c.Request.Context(), &emailports.ListEmailMessagesRequest{
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
	msg, err := h.service.GetMessage(c.Request.Context(), emailports.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, msg)
}

func (h *Handler) listSuppressions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*email.Suppression], error) {
		return h.service.ListSuppressions(
			c.Request.Context(),
			&emailports.ListEmailSuppressionsRequest{
				Filter: req,
			},
		)
	})
}

func (h *Handler) createSuppression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	suppression := new(email.Suppression)
	if err := authctx.BindJSON(c, authCtx, suppression); err != nil {
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
	helpers.JSON(c, http.StatusCreated, created)
}

func (h *Handler) deleteSuppression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.Parse(c.Param("suppressionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	err = h.service.DeleteSuppression(c.Request.Context(), emailports.GetEmailEntityRequest{
		ID:         id,
		TenantInfo: tenantInfo(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleResendWebhook(c *gin.Context) {
	token := c.Param("webhookToken")
	body, err := helpers.ReadBoundedRequestBody(c, maxWebhookBodyBytes)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo, signingSecret, err := h.service.ResolveTenantByWebhookToken(
		c.Request.Context(),
		email.ProviderResend,
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

	event, known, err := emailservice.ParseResendEvent(body, c.GetHeader("svix-id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	h.recordEvent(c, tenantInfo, event, known)
}

func (h *Handler) recordEvent(
	c *gin.Context,
	tenantInfo pagination.TenantInfo,
	event *emailservice.ProviderEvent,
	known bool,
) {
	if !known {
		c.Status(http.StatusNoContent)
		return
	}
	if err := h.service.HandleProviderEvent(c.Request.Context(), event.For(tenantInfo)); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handlePostmarkWebhook(c *gin.Context) {
	token := c.Param("webhookToken")
	body, err := helpers.ReadBoundedRequestBody(c, maxWebhookBodyBytes)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo, credentials, err := h.service.ResolveTenantByWebhookToken(
		c.Request.Context(),
		email.ProviderPostmark,
		token,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = webhooksig.VerifyBasicAuth(credentials, c.GetHeader("Authorization")); err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	event, known, err := emailservice.ParsePostmarkEvent(body)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	h.recordEvent(c, tenantInfo, event, known)
}

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
