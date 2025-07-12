package formula

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	formulaservice "github.com/emoss08/trenova/internal/core/services/formula"
	formulatypes "github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	fs *formulaservice.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	FormulaService *formulaservice.Service
	ErrorHandler   *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{fs: p.FormulaService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/formulas")

	api.Post("/test", rl.WithRateLimit(
		[]fiber.Handler{h.testFormula},
		middleware.PerMinute(60), // 60 tests per minute
	)...)
}

// * testFormula handles formula testing requests
func (h *Handler) testFormula(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(formulatypes.TestFormulaRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID
	req.UserID = reqCtx.UserID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	result, err := h.fs.TestFormula(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
