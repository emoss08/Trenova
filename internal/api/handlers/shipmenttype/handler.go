package shipmenttype

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	shipmenttypedomain "github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenttype"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	sts *shipmenttype.Service
	eh  *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	ShipmentTypeService *shipmenttype.Service
	ErrorHandler        *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{sts: p.ShipmentTypeService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipment-types")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:shipmentTypeID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:shipmentTypeID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListShipmentTypeRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			Query:  c.Query("query"),
			Limit:  c.QueryInt("limit", 10),
			Offset: c.QueryInt("offset", 0),
			TenantOpts: ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
		},
	}

	options, err := h.sts.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Next:    "",
		Prev:    "",
	})
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*shipmenttypedomain.ShipmentType], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.sts.List(fc.UserContext(), &repositories.ListShipmentTypeRequest{
			Filter: filter,
			Status: fc.Query("status"),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentTypeID, err := pulid.MustParse(c.Params("shipmentTypeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	st, err := h.sts.Get(c.UserContext(), repositories.GetShipmentTypeByIDOptions{
		ID:     shipmentTypeID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(st)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	st := new(shipmenttypedomain.ShipmentType)
	st.OrganizationID = reqCtx.OrgID
	st.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(st); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.sts.Create(c.UserContext(), st, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createEntity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentTypeID, err := pulid.MustParse(c.Params("shipmentTypeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	st := new(shipmenttypedomain.ShipmentType)
	st.ID = shipmentTypeID
	st.OrganizationID = reqCtx.OrgID
	st.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(st); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.sts.Update(c.UserContext(), st, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
