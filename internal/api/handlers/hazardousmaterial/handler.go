/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package hazardousmaterial

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	hazardousmaterialdomain "github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterial"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	hms *hazardousmaterial.Service
	eh  *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	HazardousMaterialService *hazardousmaterial.Service
	ErrorHandler             *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{hms: p.HazardousMaterialService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/hazardous-materials")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:hmID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:hmID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.LimitOffsetQueryOptions{
		TenantOpts: ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  c.QueryInt("limit", 100),
		Offset: c.QueryInt("offset", 0),
		Query:  c.Query("search"),
	}

	options, err := h.hms.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Next:    "",
		Prev:    "",
	})
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*hazardousmaterialdomain.HazardousMaterial], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.hms.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hmID, err := pulid.MustParse(c.Params("hmID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm, err := h.hms.Get(c.UserContext(), repositories.GetHazardousMaterialByIDOptions{
		ID:     hmID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hm)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm := new(hazardousmaterialdomain.HazardousMaterial)
	hm.OrganizationID = reqCtx.OrgID
	hm.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hm); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.hms.Create(c.UserContext(), hm, reqCtx.UserID)
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

	hmID, err := pulid.MustParse(c.Params("hmID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm := new(hazardousmaterialdomain.HazardousMaterial)
	hm.ID = hmID
	hm.OrganizationID = reqCtx.OrgID
	hm.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hm); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.hms.Update(c.UserContext(), hm, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
