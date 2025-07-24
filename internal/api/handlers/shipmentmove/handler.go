/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentmove

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentmove"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ShipmentMoveService *shipmentmove.Service
	ErrorHandler        *validator.ErrorHandler
}

type Handler struct {
	ss *shipmentmove.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.ShipmentMoveService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipment-moves")

	api.Post("/split/", rl.WithRateLimit(
		[]fiber.Handler{h.split},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Get("/:moveID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	moveID, err := pulid.MustParse(c.Params("moveID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	move, err := h.ss.Get(c.UserContext(), repositories.GetMoveByIDOptions{
		MoveID:            moveID,
		OrgID:             reqCtx.OrgID,
		BuID:              reqCtx.BuID,
		ExpandMoveDetails: c.QueryBool("expandMoveDetails", false),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(move)
}

func (h *Handler) split(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.SplitMoveRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	newEntity, err := h.ss.Split(c.UserContext(), req, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newEntity)
}
