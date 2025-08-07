/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package audit

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	AuditService services.AuditService
	ErrorHandler *validator.ErrorHandler
}

// Handler handles HTTP requests for audit logs
type Handler struct {
	auditService services.AuditService
	errorHandler *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		auditService: p.AuditService,
		errorHandler: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/audit-logs")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/live", h.liveStream)

	api.Get("/:auditEntryID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/resource/:resourceID", rl.WithRateLimit(
		[]fiber.Handler{h.listByResourceID},
		middleware.PerSecond(5), // 5 reads per second
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*audit.Entry], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.errorHandler.HandleError(fc, err)
		}

		return h.auditService.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.errorHandler, reqCtx, handler)
}

func (h *Handler) listByResourceID(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	resourceID, err := pulid.MustParse(c.Params("resourceID"))
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	entries, err := h.auditService.ListByResourceID(
		c.UserContext(),
		repositories.ListByResourceIDRequest{
			ResourceID: resourceID,
			OrgID:      reqCtx.OrgID,
			BuID:       reqCtx.BuID,
			UserID:     reqCtx.UserID,
		},
	)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entries)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	auditEntryID, err := pulid.MustParse(c.Params("auditEntryID"))
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	entry, err := h.auditService.GetByID(c.UserContext(), repositories.GetAuditEntryByIDOptions{
		ID:     auditEntryID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entry)
}

func (h *Handler) liveStream(c *fiber.Ctx) error {
	// CDC-based streaming only - no polling
	return h.auditService.LiveStream(c)
}
