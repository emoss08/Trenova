/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notification

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	NotificationService services.NotificationService
	ErrorHandler        *validator.ErrorHandler
}

type Handler struct {
	ns services.NotificationService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		ns: p.NotificationService,
		eh: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/notifications")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.getUserNotifications},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Post("/:notifID/read/", rl.WithRateLimit(
		[]fiber.Handler{h.markAsRead},
		middleware.PerSecond(10), // 10 writes per second
	)...)

	api.Post("/:notifID/dismiss/", rl.WithRateLimit(
		[]fiber.Handler{h.markAsDismissed},
		middleware.PerSecond(10), // 10 writes per second
	)...)

	api.Post("/read-all/", rl.WithRateLimit(
		[]fiber.Handler{h.readAllNotifications},
		middleware.PerSecond(10), // 10 writes per second
	)...)
}

func (h *Handler) getUserNotifications(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*notification.Notification], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		req := &repositories.GetUserNotificationsRequest{
			Filter:     filter,
			UnreadOnly: fc.QueryBool("unreadOnly", false),
		}
		return h.ns.GetUserNotifications(fc.UserContext(), req)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) markAsRead(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.MarkAsReadRequest
	notifID, err := pulid.MustParse(c.Params("notifID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req.NotificationID = notifID
	appctx.AddContextToRequest(reqCtx, &req)

	if err = h.ns.MarkAsRead(c.UserContext(), req); err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) markAsDismissed(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.MarkAsDismissedRequest
	notifID, err := pulid.MustParse(c.Params("notifID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req.NotificationID = notifID
	appctx.AddContextToRequest(reqCtx, &req)

	if err = h.ns.MarkAsDismissed(c.UserContext(), req); err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) readAllNotifications(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.ReadAllNotificationsRequest
	appctx.AddContextToRequest(reqCtx, &req)

	if err = h.ns.ReadAllNotifications(c.UserContext(), req); err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}
