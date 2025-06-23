package shipmentcontrol

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentcontrol"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ShipmentControlService *shipmentcontrol.Service
	ErrorHandler           *validator.ErrorHandler
}

type Handler struct {
	sc *shipmentcontrol.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{sc: p.ShipmentControlService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipment-controls")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // * 120 reads per minute
	)...)

	api.Put("/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(120), // * 120 writes per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	sc, err := h.sc.Get(c.UserContext(), &repositories.GetShipmentControlRequest{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(sc)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	sc := new(shipment.ShipmentControl)
	if err = c.BodyParser(sc); err != nil {
		return h.eh.HandleError(c, err)
	}

	sc.OrganizationID = reqCtx.OrgID
	sc.BusinessUnitID = reqCtx.BuID

	entity, err := h.sc.Update(c.UserContext(), sc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
