package dataretention

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dataretention"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Service      *dataretention.Service
	ErrorHandler *validator.ErrorHandler
}

type Handler struct {
	service *dataretention.Service
	eh      *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/data-retention")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Put("/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(120), // 120 writes per minute
	)...)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.service.Get(c.UserContext(), repositories.GetDataRetentionRequest{
		UserID: reqCtx.UserID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
	})
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

	dr := new(organization.DataRetention)
	if err = c.BodyParser(dr); err != nil {
		return h.eh.HandleError(c, err)
	}

	dr.OrganizationID = reqCtx.OrgID
	dr.BusinessUnitID = reqCtx.BuID

	entity, err := h.service.Update(c.UserContext(), dr, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
