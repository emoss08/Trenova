/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notificationpreference

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	notificationservice "github.com/emoss08/trenova/internal/core/services/notification"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	PreferenceService *notificationservice.PreferenceService
	ErrorHandler      *validator.ErrorHandler
}

type Handler struct {
	ps *notificationservice.PreferenceService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		ps: p.PreferenceService,
		eh: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/notification-preferences")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	api.Get("/user/", rl.WithRateLimit(
		[]fiber.Handler{h.userPreferences},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Get("/:preferenceID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Put("/:preferenceID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	api.Delete("/:preferenceID/", rl.WithRateLimit(
		[]fiber.Handler{h.delete},
		middleware.PerMinute(30), // 30 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*notification.NotificationPreference], error) {
		return h.ps.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	preferenceID, err := pulid.MustParse(c.Params("preferenceID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	pref, err := h.ps.GetByID(c.UserContext(), preferenceID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(pref)
}

func (h *Handler) userPreferences(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Get user ID from query param or default to current user
	userID := reqCtx.UserID
	if userIDParam := c.Query("userId"); userIDParam != "" {
		userID, err = pulid.MustParse(userIDParam)
		if err != nil {
			return h.eh.HandleError(c, err)
		}

		// Check if user is trying to access someone else's preferences
		if userID != reqCtx.UserID {
			// Check if user has manage permission
			hasManage, err := h.ps.HasManagePermission(
				c.UserContext(),
				reqCtx.UserID,
				reqCtx.OrgID,
				reqCtx.BuID,
			)
			if err != nil {
				return h.eh.HandleError(c, err)
			}

			if !hasManage {
				return h.eh.HandleError(
					c,
					errors.NewAuthorizationError(
						"You can only view your own notification preferences",
					),
				)
			}
		}
	}

	prefs, err := h.ps.GetUserPreferences(c.UserContext(), userID, reqCtx.OrgID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(prefs)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	pref := new(notification.NotificationPreference)
	pref.OrganizationID = reqCtx.OrgID
	pref.BusinessUnitID = reqCtx.BuID
	pref.UserID = reqCtx.UserID // Always set to current user

	if err = c.BodyParser(pref); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Override user ID to ensure users can only create their own preferences
	pref.UserID = reqCtx.UserID

	created, err := h.ps.Create(c.UserContext(), pref, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	preferenceID, err := pulid.MustParse(c.Params("preferenceID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// First, get the existing preference to check ownership
	existing, err := h.ps.GetByID(c.UserContext(), preferenceID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Check if user owns this preference
	if existing.UserID != reqCtx.UserID {
		return h.eh.HandleError(
			c,
			errors.NewAuthorizationError("You can only update your own notification preferences"),
		)
	}

	pref := new(notification.NotificationPreference)
	pref.ID = preferenceID
	pref.OrganizationID = reqCtx.OrgID
	pref.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(pref); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Ensure user ID cannot be changed
	pref.UserID = existing.UserID

	updated, err := h.ps.Update(c.UserContext(), pref, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updated)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	preferenceID, err := pulid.MustParse(c.Params("preferenceID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// First, get the existing preference to check ownership
	existing, err := h.ps.GetByID(c.UserContext(), preferenceID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Check if user owns this preference
	if existing.UserID != reqCtx.UserID {
		return h.eh.HandleError(
			c,
			errors.NewAuthorizationError("You can only delete your own notification preferences"),
		)
	}

	if err := h.ps.Delete(c.UserContext(), preferenceID, reqCtx.UserID); err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
