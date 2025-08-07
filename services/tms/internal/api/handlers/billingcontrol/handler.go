/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package billingcontrol

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/billingcontrol"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	BillingControlService *billingcontrol.Service
	ErrorHandler          *validator.ErrorHandler
}

type Handler struct {
	bc *billingcontrol.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{bc: p.BillingControlService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/billing-controls")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // * 120 reads per minute
	)...)

	api.Put("/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(120), // * 120 writes per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	bc, err := h.bc.Get(c.UserContext(), &repositories.GetBillingControlRequest{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(bc)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	bc := new(billing.BillingControl)
	bc.OrganizationID = reqCtx.OrgID
	bc.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(bc); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.bc.Update(c.UserContext(), bc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
