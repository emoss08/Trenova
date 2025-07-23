// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package hazmatsegregationrule

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	hazmatsegdomain "github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	HazmatSegregationRuleService *hazmatsegregationrule.Service
	ErrorHandler                 *validator.ErrorHandler
}

type Handler struct {
	hsrs *hazmatsegregationrule.Service
	eh   *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{hsrs: p.HazmatSegregationRuleService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/hazmat-segregation-rules")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:hsrID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:hsrID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*hazmatsegdomain.HazmatSegregationRule], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.hsrs.List(fc.UserContext(), &repositories.ListHazmatSegregationRuleRequest{
			Filter:                 filter,
			IncludeHazmatMaterials: c.QueryBool("includeHazmatMaterials", false),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hsrID, err := pulid.MustParse(c.Params("hsrID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hsr, err := h.hsrs.Get(c.UserContext(), &repositories.GetHazmatSegregationRuleByIDRequest{
		ID:     hsrID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
		// IncludeHazmatMaterials: c.QueryBool("includeHazmatMaterials", false),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hsr)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hsr := new(hazmatsegdomain.HazmatSegregationRule)
	hsr.OrganizationID = reqCtx.OrgID
	hsr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hsr); err != nil {
		return h.eh.HandleError(c, err)
	}

	createdHSR, err := h.hsrs.Create(c.UserContext(), hsr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createdHSR)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hsrID, err := pulid.MustParse(c.Params("hsrID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hsr := new(hazmatsegdomain.HazmatSegregationRule)
	hsr.ID = hsrID
	hsr.OrganizationID = reqCtx.OrgID
	hsr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hsr); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedHSR, err := h.hsrs.Update(c.UserContext(), hsr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedHSR)
}
