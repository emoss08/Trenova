/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package formula

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	formulaservice "github.com/emoss08/trenova/internal/core/services/formula"
	formulatypes "github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	fs *formulaservice.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	FormulaService *formulaservice.Service
	ErrorHandler   *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{fs: p.FormulaService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/formula-templates")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/:formulaTemplateID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/test", rl.WithRateLimit(
		[]fiber.Handler{h.testFormula},
		middleware.PerMinute(60), // 60 tests per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*formulatemplate.FormulaTemplate], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.fs.List(fc.UserContext(), &repositories.ListFormulaTemplateOptions{
			Filter: filter,
			FormulaTemplateOptions: repositories.FormulaTemplateOptions{
				IncludeInactive: fc.QueryBool("includeInactive", false),
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

	ftID, err := pulid.MustParse(c.Params("formulaTemplateID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	ft, err := h.fs.Get(c.UserContext(), &repositories.GetFormulaTemplateByIDOptions{
		ID:     ftID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ft)
}

// * testFormula handles formula testing requests
func (h *Handler) testFormula(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(formulatypes.TestFormulaRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID
	req.UserID = reqCtx.UserID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	result, err := h.fs.TestFormula(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
