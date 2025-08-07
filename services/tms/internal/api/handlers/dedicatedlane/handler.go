/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dedicatedlane

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	dedicatedlanedomain "github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ds *dedicatedlane.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	DedicatedLaneService *dedicatedlane.Service
	ErrorHandler         *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ds: p.DedicatedLaneService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	dedicatedLaneAPI := r.Group("/dedicated-lanes")

	dedicatedLaneAPI.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	dedicatedLaneAPI.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	dedicatedLaneAPI.Post("/find-by-shipment", rl.WithRateLimit(
		[]fiber.Handler{h.findByShipment},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	dedicatedLaneAPI.Get("/:dedicatedLaneID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	dedicatedLaneAPI.Put("/:dedicatedLaneID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*dedicatedlanedomain.DedicatedLane], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ds.List(fc.UserContext(), &repositories.ListDedicatedLaneRequest{
			Filter: filter,
			FilterOptions: repositories.DedicatedLaneFilterOptions{
				ExpandDetails: fc.QueryBool("expandDetails", false),
			},
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	dedicatedLaneID, err := pulid.MustParse(c.Params("dedicatedLaneID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	r, err := h.ds.Get(c.UserContext(), &repositories.GetDedicatedLaneByIDRequest{
		ID:     dedicatedLaneID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(r)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	dl := new(dedicatedlanedomain.DedicatedLane)
	dl.OrganizationID = reqCtx.OrgID
	dl.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(dl); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ds.Create(c.UserContext(), dl, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) findByShipment(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.FindDedicatedLaneByShipmentRequest)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.OrganizationID = reqCtx.OrgID
	req.BusinessUnitID = reqCtx.BuID

	dl, err := h.ds.FindByShipment(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(dl)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	dedicatedLaneID, err := pulid.MustParse(c.Params("dedicatedLaneID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	dl := new(dedicatedlanedomain.DedicatedLane)
	dl.ID = dedicatedLaneID
	dl.OrganizationID = reqCtx.OrgID
	dl.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(dl); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedDedicatedLane, err := h.ds.Update(c.UserContext(), dl, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedDedicatedLane)
}
