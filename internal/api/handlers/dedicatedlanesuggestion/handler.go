package dedicatedlanesuggestion

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	dlservice "github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Logger            *logger.Logger
	SuggestionService *dlservice.SuggestionService
	ErrorHandler      *validator.ErrorHandler
}

type Handler struct {
	l  *zerolog.Logger
	ss *dlservice.SuggestionService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	log := p.Logger.With().
		Str("handler", "dedicated_lane_suggestion").
		Logger()

	return &Handler{
		l:  &log,
		ss: p.SuggestionService,
		eh: p.ErrorHandler,
	}
}

// RegisterRoutes registers the dedicated lane suggestion routes
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	suggestionAPI := r.Group("/dedicated-lane-suggestions")

	suggestionAPI.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.List},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	suggestionAPI.Get("/:id", rl.WithRateLimit(
		[]fiber.Handler{h.Get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	suggestionAPI.Post("/:id/accept", rl.WithRateLimit(
		[]fiber.Handler{h.AcceptSuggestion},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	suggestionAPI.Post("/:id/reject", rl.WithRateLimit(
		[]fiber.Handler{h.RejectSuggestion},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	suggestionAPI.Post("/analyze-patterns", rl.WithRateLimit(
		[]fiber.Handler{h.AnalyzePatterns},
		middleware.PerMinute(10), // 10 analysis requests per minute (expensive operation)
	)...)

	suggestionAPI.Post("/expire-old", rl.WithRateLimit(
		[]fiber.Handler{h.ExpireOldSuggestions},
		middleware.PerMinute(5), // 5 expire operations per minute
	)...)
}

// List returns a paginated list of dedicated lane suggestions
func (h *Handler) List(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion], error) {
		req := new(repositories.ListDedicatedLaneSuggestionRequest)

		if err = fc.QueryParser(req); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		req.Filter = filter

		return h.ss.List(fc.UserContext(), req)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

// Get returns a single dedicated lane suggestion by ID
func (h *Handler) Get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	idParam := c.Params("id")
	id, err := pulid.Parse(idParam)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:     id,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	}

	suggestion, err := h.ss.Get(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(suggestion)
}

// AcceptSuggestion accepts a suggestion and creates a dedicated lane
func (h *Handler) AcceptSuggestion(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	idParam := c.Params("id")
	suggestionID, err := pulid.Parse(idParam)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(dedicatedlane.SuggestionAcceptRequest)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Set required fields from URL and context
	req.SuggestionID = suggestionID
	req.OrganizationID = reqCtx.OrgID
	req.BusinessUnitID = reqCtx.BuID
	req.ProcessedByID = reqCtx.UserID

	dedicatedLane, err := h.ss.AcceptSuggestion(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dedicatedLane)
}

// RejectSuggestion rejects a suggestion
func (h *Handler) RejectSuggestion(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	idParam := c.Params("id")
	suggestionID, err := pulid.Parse(idParam)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(dedicatedlane.SuggestionRejectRequest)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Set required fields from URL and context
	req.SuggestionID = suggestionID
	req.OrganizationID = reqCtx.OrgID
	req.BusinessUnitID = reqCtx.BuID
	req.ProcessedByID = reqCtx.UserID

	suggestion, err := h.ss.RejectSuggestion(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(suggestion)
}

// AnalyzePatterns triggers pattern analysis and suggestion creation
func (h *Handler) AnalyzePatterns(c *fiber.Ctx) error {
	req := new(dedicatedlane.PatternAnalysisRequest)
	if err := c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	result, err := h.ss.AnalyzePatterns(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(result)
}

// ExpireOldSuggestions expires old suggestions for the organization
func (h *Handler) ExpireOldSuggestions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	expired, err := h.ss.ExpireOldSuggestions(c.UserContext(), reqCtx.OrgID, reqCtx.BuID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(fiber.Map{
		"expiredCount": expired,
	})
}
