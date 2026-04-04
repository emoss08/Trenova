package notificationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *notificationservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *notificationservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	notifs := rg.Group("/notifications")
	notifs.GET("/", h.list)
	notifs.GET("/unread-count", h.unreadCount)
	notifs.PATCH("/mark-read", h.markRead)
	notifs.PATCH("/mark-all-read", h.markAllRead)
}

func (h *Handler) list(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, auth)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*notification.Notification], error) {
			return h.service.List(c.Request.Context(), &repositories.ListNotificationsRequest{
				Filter: req,
			})
		},
	)
}

func (h *Handler) unreadCount(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	count, err := h.service.CountUnread(c.Request.Context(), auth.UserID, pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

type markReadRequest struct {
	IDs []pulid.ID `json:"ids"`
}

func (h *Handler) markRead(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	var req markReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids list cannot be empty"})
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), repositories.MarkNotificationsReadRequest{
		IDs: req.IDs,
		TenantInfo: pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) markAllRead(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	if err := h.service.MarkAllAsRead(c.Request.Context(), auth.UserID, pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
