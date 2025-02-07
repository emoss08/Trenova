package assignment

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/assignment"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	AssignmentService *assignment.Service
	ErrorHandler      *validator.ErrorHandler
}

type Handler struct {
	as *assignment.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		as: p.AssignmentService,
		eh: p.ErrorHandler,
	}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/assignments")
	api.Post("/single/", rl.WithRateLimit(
		[]fiber.Handler{h.assign},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h Handler) assign(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	amt := new(shipment.Assignment)
	amt.OrganizationID = reqCtx.OrgID
	amt.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(amt); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.as.SingleAssign(c.UserContext(), amt, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
