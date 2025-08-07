/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package location

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	locationdomain "github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/location"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ls *location.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	LocationService *location.Service
	ErrorHandler    *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ls: p.LocationService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/locations")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(500), // 500 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(300), // 300 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(300), // 300 writes per minute
	)...)

	api.Get("/:locationID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Put("/:locationID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(120), // 120 writes per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListLocationOptions{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  c.QueryInt("limit", 100),
			Offset: c.QueryInt("offset", 0),
			Query:  c.Query("search"),
		},
	}

	options, err := h.ls.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Count:   len(options),
		Next:    "",
		Prev:    "",
	})
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*locationdomain.Location], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ls.List(fc.UserContext(), &repositories.ListLocationOptions{
			Filter:          filter,
			IncludeCategory: c.QueryBool("includeCategory"),
			IncludeState:    c.QueryBool("includeState"),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	locationID, err := pulid.MustParse(c.Params("locationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	com, err := h.ls.Get(c.UserContext(), repositories.GetLocationByIDOptions{
		ID:              locationID,
		BuID:            reqCtx.BuID,
		OrgID:           reqCtx.OrgID,
		UserID:          reqCtx.UserID,
		IncludeCategory: c.QueryBool("includeCategory"),
		IncludeState:    c.QueryBool("includeState"),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(com)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	loc := new(locationdomain.Location)
	loc.OrganizationID = reqCtx.OrgID
	loc.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(loc); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.ls.Create(c.UserContext(), loc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createEntity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	locationID, err := pulid.MustParse(c.Params("locationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	loc := new(locationdomain.Location)
	loc.ID = locationID
	loc.OrganizationID = reqCtx.OrgID
	loc.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(loc); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.ls.Update(c.UserContext(), loc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
