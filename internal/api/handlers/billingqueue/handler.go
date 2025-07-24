/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package billingqueue

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	queuedomain "github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/billingqueue"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Service      *billingqueue.Service
	ErrorHandler *validator.ErrorHandler
}

type Handler struct {
	s  *billingqueue.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		s:  p.Service,
		eh: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/billing-queue")
	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5),
	)...)
	api.Get("/:billingQueueID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerSecond(5),
	)...)
	api.Post("/bulk-transfer", rl.WithRateLimit(
		[]fiber.Handler{h.bulkTransfer},
		middleware.PerMinute(5), // 5 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*queuedomain.QueueItem], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.s.List(fc.UserContext(), &repositories.ListBillingQueueRequest{
			Filter: filter,
			FilterOptions: repositories.BillingQueueFilterOptions{
				IncludeShipmentDetails: fc.QueryBool("includeShipmentDetails"),
				Status:                 fc.Query("status"),
				BillType:               fc.Query("billType"),
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

	billingQueueID, err := pulid.MustParse(c.Params("billingQueueID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	qi, err := h.s.Get(c.UserContext(), &repositories.GetBillingQueueItemRequest{
		BillingQueueItemID: billingQueueID,
		OrgID:              reqCtx.OrgID,
		BuID:               reqCtx.BuID,
		UserID:             reqCtx.UserID,
		FilterOptions: repositories.BillingQueueFilterOptions{
			IncludeShipmentDetails: c.QueryBool("includeShipmentDetails"),
			Status:                 c.Query("status"),
			BillType:               c.Query("billType"),
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(qi)
}

func (h *Handler) bulkTransfer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.s.BulkTransfer(c.UserContext(), &repositories.BulkTransferRequest{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
