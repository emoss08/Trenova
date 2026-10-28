package inboxpresencehandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const Scope = "inbox:messages"

type Params struct {
	fx.In

	Gateway              realtimeports.RealtimeGateway
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	gateway realtimeports.RealtimeGateway
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		gateway: p.Gateway,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/inbox/presence")
	canRead := h.pm.RequirePermission(permission.ResourceInboundMessage.String(), permission.OpRead)
	group.POST("/", canRead, h.join)
	group.DELETE("/", canRead, h.leave)
}

type presenceRequest struct {
	ConnectionID string  `json:"connectionId"`
	MessageID    *string `json:"messageId"`
	Activity     *string `json:"activity"`
}

// @Summary Join or update Triage presence
// @Description Joins the tenant's Triage presence scope, or changes which message the caller is on
// @Description and whether they are viewing it or writing a reply, and returns everyone in it.
// @ID joinInboxPresence
// @Tags Realtime
// @Accept json
// @Produce json
// @Param request body presenceRequest true "Live connection and what the caller is doing"
// @Success 200 {object} services.RealtimePresenceSnapshot
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Router /inbox/presence/ [post]
func (h *Handler) join(c *gin.Context) {
	var body presenceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := scopeRequest(c, body.ConnectionID)
	req.State = realtimeports.RealtimePresenceState{
		MessageID: strings.TrimSpace(stringutils.FromPtr(body.MessageID)),
		Activity:  strings.TrimSpace(stringutils.FromPtr(body.Activity)),
	}

	snapshot, err := h.gateway.JoinPresence(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, snapshot)
}

// @Summary Leave Triage presence
// @ID leaveInboxPresence
// @Tags Realtime
// @Param connectionId query string true "Live connection"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Router /inbox/presence/ [delete]
func (h *Handler) leave(c *gin.Context) {
	if err := h.gateway.LeavePresence(
		c.Request.Context(),
		scopeRequest(c, c.Query("connectionId")),
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func scopeRequest(c *gin.Context, connectionID string) *realtimeports.RealtimeScopeRequest {
	authCtx := authctx.GetAuthContext(c)

	return &realtimeports.RealtimeScopeRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		ConnectionID:   connectionID,
		Scope:          Scope,
	}
}
