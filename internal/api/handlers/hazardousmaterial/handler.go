package hazardousmaterial

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/api/middleware"
	hazardousmaterialdomain "github.com/trenova-app/transport/internal/core/domain/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/services/hazardousmaterial"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type Handler struct {
	hms *hazardousmaterial.Service
	eh  *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	HazardousMaterialService *hazardousmaterial.Service
	ErrorHandler             *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{hms: p.HazardousMaterialService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/hazardous-materials")

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

	api.Get("/:hmID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:hmID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.LimitOffsetQueryOptions{
		TenantOpts: &ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  c.QueryInt("limit", 100),
		Offset: c.QueryInt("offset", 0),
		Query:  c.Query("search"),
	}

	options, err := h.hms.SelectOptions(c.UserContext(), opts)
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

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*hazardousmaterialdomain.HazardousMaterial], error) {
		return h.hms.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hmID, err := pulid.MustParse(c.Params("hmID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm, err := h.hms.Get(c.UserContext(), repositories.GetHazardousMaterialByIDOptions{
		ID:    hmID,
		BuID:  reqCtx.BuID,
		OrgID: reqCtx.OrgID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hm)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm := new(hazardousmaterialdomain.HazardousMaterial)
	hm.OrganizationID = reqCtx.OrgID
	hm.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hm); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.hms.Create(c.UserContext(), hm, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createEntity)
}

func (h Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hmID, err := pulid.MustParse(c.Params("hmID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	hm := new(hazardousmaterialdomain.HazardousMaterial)
	hm.ID = hmID
	hm.OrganizationID = reqCtx.OrgID
	hm.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(hm); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.hms.Update(c.UserContext(), hm, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
