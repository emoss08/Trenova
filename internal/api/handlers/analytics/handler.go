/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package analytics

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/analytics"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	AnalyticsService services.AnalyticsService
	ErrorHandler     *validator.ErrorHandler
}

// Handler handles HTTP requests for analytics
type Handler struct {
	analyticsService services.AnalyticsService
	errorHandler     *validator.ErrorHandler
}

// NewHandler creates a new analytics handler
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		analyticsService: p.AnalyticsService,
		errorHandler:     p.ErrorHandler,
	}
}

// RegisterRoutes registers analytics routes
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/analytics")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.getAnalytics},
		middleware.PerSecond(5), // 5 reads per second
	)...)
}

// getAnalytics handles GET requests to fetch analytics data
func (h *Handler) getAnalytics(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Parse query parameters
	req := new(analytics.Request)
	if err = c.QueryParser(req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Validate page parameter
	if req.Page == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"page",
			"required",
			"Page parameter is required",
		))
	}

	// Set up options for analytics service
	opts := &services.AnalyticsRequestOptions{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
		Page:   req.Page,
		Limit:  req.Limit,
	}

	// Set date range if provided
	if req.StartDate > 0 && req.EndDate > 0 {
		opts.DateRange = &services.DateRange{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
		}
	}

	// Get analytics data
	data, err := h.analyticsService.GetAnalytics(c.UserContext(), opts)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(data)
}
