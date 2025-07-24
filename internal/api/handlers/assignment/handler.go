/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package assignment

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/assignment"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	AssignmentService *assignment.Service
	ErrorHandler      *validator.ErrorHandler
}

type Handler struct {
	as *assignment.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		as: p.AssignmentService,
		eh: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/assignments")
	api.Post("/single/", rl.WithRateLimit(
		[]fiber.Handler{h.assign},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Post("/bulk/", rl.WithRateLimit(
		[]fiber.Handler{h.bulkAssign},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Put("/:assignmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.reassign},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:assignmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	assignmentID, err := pulid.MustParse(c.Params("assignmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.as.Get(c.UserContext(), repositories.GetAssignmentByIDOptions{
		ID:     assignmentID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) assign(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	amt := new(shipment.Assignment)
	amt.OrganizationID = reqCtx.OrgID
	amt.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(amt); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.as.SingleAssign(c.UserContext(), amt, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) reassign(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	assignmentID, err := pulid.MustParse(c.Params("assignmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	amt := new(shipment.Assignment)
	amt.ID = assignmentID
	amt.OrganizationID = reqCtx.OrgID
	amt.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(amt); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.as.Reassign(c.UserContext(), amt, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) bulkAssign(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	amt := new(repositories.AssignmentRequest)
	amt.OrgID = reqCtx.OrgID
	amt.BuID = reqCtx.BuID

	if err = c.BodyParser(amt); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.as.BulkAssign(c.UserContext(), amt)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
