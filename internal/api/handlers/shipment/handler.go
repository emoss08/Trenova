package shipment

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipment"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ss *shipment.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	ShipmentService *shipment.Service
	ErrorHandler    *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.ShipmentService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipments")

	api.Get("/select-options", rl.WithRateLimit(
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

	api.Get("/:shipmentID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:shipmentID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListShipmentOptions{
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

	options, err := h.ss.SelectOptions(c.UserContext(), opts)
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

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*shipmentdomain.Shipment], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ss.List(fc.UserContext(), &repositories.ListShipmentOptions{
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: c.QueryBool("expandShipmentDetails"),
			},
			Filter: filter,
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp, err := h.ss.Get(c.UserContext(), repositories.GetShipmentByIDOptions{
		ID:     shipmentID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: c.QueryBool("expandShipmentDetails"),
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(shp)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp := new(shipmentdomain.Shipment)
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ss.Create(c.UserContext(), shp, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shpID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp := new(shipmentdomain.Shipment)
	shp.ID = shpID
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ss.Update(c.UserContext(), shp, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
