// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
		[]fiber.Handler{h.list},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	suggestionAPI.Get("/:suggestionID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	suggestionAPI.Post("/:suggestionID/accept", rl.WithRateLimit(
		[]fiber.Handler{h.accept},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	suggestionAPI.Post("/:suggestionID/reject", rl.WithRateLimit(
		[]fiber.Handler{h.reject},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	suggestionAPI.Post("/analyze-patterns", rl.WithRateLimit(
		[]fiber.Handler{h.analyzePatterns},
		middleware.PerMinute(10), // 10 analysis requests per minute (expensive operation)
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
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

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	suggestionID, err := pulid.MustParse(c.Params("suggestionID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.GetDedicatedLaneSuggestionByIDRequest{
		ID:     suggestionID,
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

func (h *Handler) accept(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	suggestionID, err := pulid.MustParse(c.Params("suggestionID"))
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

func (h *Handler) reject(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	suggestionID, err := pulid.MustParse(c.Params("suggestionID"))
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

func (h *Handler) analyzePatterns(c *fiber.Ctx) error {
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
