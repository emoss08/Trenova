package shipmentmove

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentmove"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ShipmentMoveService *shipmentmove.Service
	ErrorHandler        *validator.ErrorHandler
}

type Handler struct {
	ss *shipmentmove.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.ShipmentMoveService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipment-moves")

	api.Post("/split/", rl.WithRateLimit(
		[]fiber.Handler{h.split},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h Handler) split(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.SplitMoveRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	newEntity, err := h.ss.Split(c.UserContext(), req, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newEntity)
}
