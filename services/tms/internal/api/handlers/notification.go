package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type NotificationHandlerParams struct {
	fx.In

	Service      services.NotificationService
	ErrorHandler *helpers.ErrorHandler
}

type NotificationHandler struct {
	service services.NotificationService
	eh      *helpers.ErrorHandler
}

func NewNotificationHandler(p NotificationHandlerParams) *NotificationHandler {
	return &NotificationHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *NotificationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/notifications/")
	api.GET("", h.list)
	api.POST(":id/read/", h.markAsRead)
	api.POST(":id/dismiss/", h.markAsDismissed)
	api.POST("read-all/", h.markAllAsRead)
}

func (h *NotificationHandler) list(c *gin.Context) {
	pagination.Handle[*notification.Notification](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(
			func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*notification.Notification], error) {
				return h.service.GetUserNotifications(
					c.Request.Context(),
					&repositories.GetUserNotificationsRequest{
						Filter:     opts,
						UnreadOnly: helpers.QueryBool(c, "unreadOnly", false),
					},
				)
			},
		)
}

func (h *NotificationHandler) markAsRead(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.MarkAsReadRequest
	notifID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.NotificationID = notifID
	context.AddContextToRequest(authCtx, &req)

	if err = h.service.MarkAsRead(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *NotificationHandler) markAsDismissed(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.MarkAsDismissedRequest
	notifID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.NotificationID = notifID
	context.AddContextToRequest(authCtx, &req)

	if err = h.service.MarkAsDismissed(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *NotificationHandler) markAllAsRead(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req repositories.ReadAllNotificationsRequest
	context.AddContextToRequest(authCtx, &req)

	if err := h.service.ReadAllNotifications(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
