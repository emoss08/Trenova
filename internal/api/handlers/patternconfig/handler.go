package patternconfig

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	dlDomain "github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ErrorHandler         *validator.ErrorHandler
	PatternConfigService *dedicatedlane.PatternService
}

type Handler struct {
	eh *validator.ErrorHandler
	pc *dedicatedlane.PatternService
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		eh: p.ErrorHandler,
		pc: p.PatternConfigService,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/pattern-config")

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

	pc, err := h.pc.GetPatternConfig(c.UserContext(), repositories.GetPatternConfigRequest{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(pc)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	pc := new(dlDomain.PatternConfig)
	if err = c.BodyParser(pc); err != nil {
		return h.eh.HandleError(c, err)
	}

	pc.OrganizationID = reqCtx.OrgID
	pc.BusinessUnitID = reqCtx.BuID

	updatedPC, err := h.pc.UpdatePatternConfig(c.UserContext(), pc, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedPC)
}
