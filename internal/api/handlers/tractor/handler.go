package tractor

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	tractordomain "github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tractor"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ts *tractor.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	TractorService *tractor.Service
	ErrorHandler   *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ts: p.TractorService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/tractors")

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Get("/:tractorID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:tractorID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:tractorID/assignment/", rl.WithRateLimit(
		[]fiber.Handler{h.assignment},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := &repositories.ListTractorRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  100,
			Offset: 0,
		},
	}

	options, err := h.ts.SelectOptions(c.UserContext(), req)
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

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*tractordomain.Tractor], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ts.List(fc.UserContext(), &repositories.ListTractorRequest{
			Filter: filter,
			FilterOptions: repositories.TractorFilterOptions{
				Status:                  fc.Query("status"),
				IncludeWorkerDetails:    fc.QueryBool("includeWorkerDetails"),
				IncludeEquipmentDetails: fc.QueryBool("includeEquipmentDetails"),
				IncludeFleetDetails:     fc.QueryBool("includeFleetDetails"),
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

	tractorID, err := pulid.MustParse(c.Params("tractorID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tr, err := h.ts.Get(c.UserContext(), &repositories.GetTractorByIDRequest{
		TractorID: tractorID,
		BuID:      reqCtx.BuID,
		OrgID:     reqCtx.OrgID,
		UserID:    reqCtx.UserID,
		FilterOptions: repositories.TractorFilterOptions{
			IncludeWorkerDetails:    c.QueryBool("includeWorkerDetails"),
			IncludeEquipmentDetails: c.QueryBool("includeEquipmentDetails"),
			IncludeFleetDetails:     c.QueryBool("includeFleetDetails"),
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(tr)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tr := new(tractordomain.Tractor)
	tr.OrganizationID = reqCtx.OrgID
	tr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(tr); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ts.Create(c.UserContext(), tr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	trID, err := pulid.MustParse(c.Params("tractorID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tr := new(tractordomain.Tractor)
	tr.ID = trID
	tr.OrganizationID = reqCtx.OrgID
	tr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(tr); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ts.Update(c.UserContext(), tr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) assignment(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tractorID, err := pulid.MustParse(c.Params("tractorID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	assignment, err := h.ts.Assignment(c.UserContext(), repositories.TractorAssignmentRequest{
		TractorID: tractorID,
		OrgID:     reqCtx.OrgID,
		BuID:      reqCtx.BuID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(assignment)
}
