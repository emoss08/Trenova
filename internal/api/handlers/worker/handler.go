package worker

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	workerdomain "github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/worker"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ws *worker.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	WorkerService *worker.Service
	ErrorHandler  *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ws: p.WorkerService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/workers")

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

	api.Get("/:workerID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:workerID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListWorkerOptions{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  100,
			Offset: 0,
		},
	}

	options, err := h.ws.SelectOptions(c.UserContext(), opts)
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

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*workerdomain.Worker], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ws.List(fc.UserContext(), &repositories.ListWorkerOptions{
			Filter:         filter,
			IncludeProfile: c.QueryBool("includeProfile"),
			IncludePTO:     c.QueryBool("includePTO"),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	workerID, err := pulid.MustParse(c.Params("workerID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	wrk, err := h.ws.Get(c.UserContext(), repositories.GetWorkerByIDOptions{
		WorkerID:       workerID,
		BuID:           reqCtx.BuID,
		OrgID:          reqCtx.OrgID,
		UserID:         reqCtx.UserID,
		IncludeProfile: c.QueryBool("includeProfile"),
		IncludePTO:     c.QueryBool("includePTO"),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(wrk)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	wkr := new(workerdomain.Worker)
	wkr.OrganizationID = reqCtx.OrgID
	wkr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(wkr); err != nil {
		return h.eh.HandleError(c, err)
	}

	createdWorker, err := h.ws.Create(c.UserContext(), wkr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createdWorker)
}

func (h Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	wkrID, err := pulid.MustParse(c.Params("workerID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	wkr := new(workerdomain.Worker)
	wkr.ID = wkrID
	wkr.OrganizationID = reqCtx.OrgID
	wkr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(wkr); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedWorker, err := h.ws.Update(c.UserContext(), wkr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedWorker)
}
