/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package holdreason

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/holdreason"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Service      *holdreason.Service
	ErrorHandler *validator.ErrorHandler
}

type Handler struct {
	hrs *holdreason.Service
	eh  *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		hrs: p.Service,
		eh:  p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/hold-reasons")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/:hrID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Put("/:hrID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	eo, err := paginationutils.ParseEnhancedQueryFromJSON(c, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	qo := new(repositories.ListHoldReasonRequest)
	if err = paginationutils.ParseAdditionalQueryParams(c, qo); err != nil {
		return h.eh.HandleError(c, err)
	}

	listOpts := repositories.BuildHoldReasonListOptions(eo)

	handler := func(fc *fiber.Ctx, filter *ports.QueryOptions) (*ports.ListResult[*shipment.HoldReason], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.hrs.List(fc.UserContext(), listOpts)
	}

	return limitoffsetpagination.HandleEnhancedPaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hrID, err := pulid.MustParse(c.Params("hrID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hr, err := h.hrs.Get(c.UserContext(), &repositories.GetHoldReasonByIDRequest{
		ID:     hrID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hr)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hr := new(shipment.HoldReason)
	hr.OrganizationID = reqCtx.OrgID
	hr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hr); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.hrs.Create(c.UserContext(), hr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hrID, err := pulid.MustParse(c.Params("hrID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hr := new(shipment.HoldReason)
	hr.ID = hrID
	hr.OrganizationID = reqCtx.OrgID
	hr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hr); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedHR, err := h.hrs.Update(c.UserContext(), hr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedHR)
}
