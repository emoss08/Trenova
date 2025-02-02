package fleetcode

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	fleetcodedomain "github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fleetcode"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	fs *fleetcode.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	FleetCodeService *fleetcode.Service
	ErrorHandler     *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{fs: p.FleetCodeService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/fleet-codes")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:fleetCodeID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:fleetCodeID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListFleetCodeOptions{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  c.QueryInt("limit", 100),
			Offset: c.QueryInt("offset", 0),
		},
	}

	options, err := h.fs.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Next:    "",
		Prev:    "",
	})
}

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*fleetcodedomain.FleetCode], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.fs.List(fc.UserContext(), &repositories.ListFleetCodeOptions{
			Filter:                filter,
			IncludeManagerDetails: c.QueryBool("includeManagerDetails"),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fleetCodeID, err := pulid.MustParse(c.Params("fleetCodeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fc, err := h.fs.Get(c.UserContext(), repositories.GetFleetCodeByIDOptions{
		ID:                    fleetCodeID,
		BuID:                  reqCtx.BuID,
		OrgID:                 reqCtx.OrgID,
		UserID:                reqCtx.UserID,
		IncludeManagerDetails: c.QueryBool("includeManagerDetails"),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fc)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fc := new(fleetcodedomain.FleetCode)
	fc.OrganizationID = reqCtx.OrgID
	fc.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(fc); err != nil {
		return h.eh.HandleError(c, err)
	}

	createdFleetCode, err := h.fs.Create(c.UserContext(), fc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createdFleetCode)
}

func (h Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fleetCodeID, err := pulid.MustParse(c.Params("fleetCodeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fc := new(fleetcodedomain.FleetCode)
	fc.ID = fleetCodeID
	fc.OrganizationID = reqCtx.OrgID
	fc.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(fc); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedFleetCode, err := h.fs.Update(c.UserContext(), fc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedFleetCode)
}
