/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package customer

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	customerdomain "github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customer"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	cs *customer.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	CustomerService *customer.Service
	ErrorHandler    *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{cs: p.CustomerService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/customers")

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

	api.Get("/:customerID/document-requirements", rl.WithRateLimit(
		[]fiber.Handler{h.getDocumentRequirements},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/:customerID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:customerID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListCustomerOptions{
		Filter: &ports.QueryOptions{
			Limit:  c.QueryInt("limit", 100),
			Offset: c.QueryInt("offset", 0),
			Query:  c.Query("search"),
			TenantOpts: ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
		},
	}

	options, err := h.cs.SelectOptions(c.UserContext(), opts)
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

	// Parse enhanced query parameters using the helper
	enhancedOpts, err := paginationutils.ParseEnhancedQueryFromJSON(c, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse additional customer-specific options
	qo := new(repositories.ListCustomerOptions)
	if err = paginationutils.ParseAdditionalQueryParams(c, qo); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Build list options using the helper
	listOpts := repositories.BuildCustomerListOptions(enhancedOpts, qo)

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*customerdomain.Customer], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.cs.List(c.UserContext(), listOpts)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	customerID, err := pulid.MustParse(c.Params("customerID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.cs.Get(c.UserContext(), repositories.GetCustomerByIDOptions{
		ID:                    customerID,
		BuID:                  reqCtx.BuID,
		OrgID:                 reqCtx.OrgID,
		UserID:                reqCtx.UserID,
		IncludeState:          c.QueryBool("includeState"),
		IncludeBillingProfile: c.QueryBool("includeBillingProfile"),
		IncludeEmailProfile:   c.QueryBool("includeEmailProfile"),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) getDocumentRequirements(c *fiber.Ctx) error {
	customerID, err := pulid.MustParse(c.Params("customerID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	requirements, err := h.cs.GetDocumentRequirements(c.UserContext(), customerID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(requirements)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity := new(customerdomain.Customer)
	entity.OrganizationID = reqCtx.OrgID
	entity.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(entity); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.cs.Create(c.UserContext(), entity, reqCtx.UserID)
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

	customerID, err := pulid.MustParse(c.Params("customerID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity := new(customerdomain.Customer)
	entity.ID = customerID
	entity.OrganizationID = reqCtx.OrgID
	entity.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(entity); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.cs.Update(c.UserContext(), entity, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
