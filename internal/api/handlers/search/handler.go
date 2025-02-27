package search

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/services/search"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// HandlerParams contains dependencies for the search handler
type HandlerParams struct {
	fx.In

	SearchService *search.Service
	ErrorHandler  *validator.ErrorHandler
	Logger        *logger.Logger
}

// Handler manages HTTP requests for search operations
type Handler struct {
	ss  *search.Service
	eh  *validator.ErrorHandler
	log *zerolog.Logger
}

// NewHandler creates a new search handler
func NewHandler(p HandlerParams) *Handler {
	log := p.Logger.With().Str("component", "search_handler").Logger()

	return &Handler{
		ss:  p.SearchService,
		eh:  p.ErrorHandler,
		log: &log,
	}
}

// RegisterRoutes sets up routing for search endpoints
func (h *Handler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/search")

	// Basic search endpoint
	api.Get("/", h.Search)

	// Advanced search with POST for complex queries
	api.Post("/", h.AdvancedSearch)

	// Type-specific search endpoints
	api.Get("/shipments", h.SearchShipments)
	api.Get("/equipment", h.SearchEquipment)
	api.Get("/customers", h.SearchCustomers)
	api.Get("/drivers", h.SearchDrivers)

	// Suggest endpoint for autocomplete
	api.Get("/suggest", h.Suggest)

	// Health check endpoint
	api.Get("/health", h.GetHealth)
}

// Search handles GET search requests
func (h *Handler) Search(c *fiber.Ctx) error {
	start := time.Now()

	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse and validate query parameters
	var req Request
	if err = c.QueryParser(&req); err != nil {
		h.log.Error().Err(err).Msg("failed to parse search query parameters")
		return h.eh.HandleError(c, errors.NewValidationError("query", errors.ErrInvalid, "invalid query parameters"))
	}

	// Apply default values
	if req.Limit <= 0 {
		req.Limit = 20
	} else if req.Limit > 100 {
		req.Limit = 100
	}

	// Require a search query
	if strings.TrimSpace(req.Query) == "" {
		return h.eh.HandleError(c, errors.NewValidationError("q", errors.ErrRequired, "search query is required"))
	}

	// Prepare sort options
	sortOptions := []string{"createdAt:desc"} // Default sort
	if req.SortBy != "" {
		sortOptions = []string{req.SortBy}
	}

	// Execute search
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Facets:    req.Facets,
		SortBy:    sortOptions,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		h.log.Error().Err(err).Interface("request", searchReq).Msg("search execution failed")
		return h.eh.HandleError(c, err)
	}

	// Build response
	resp := Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
		Facets:      results.Facets,
	}

	// Add metadata for pagination
	resp.Metadata = map[string]any{
		"hasMore": (req.Offset + req.Limit) < results.Total,
		"nextOffset": func() int {
			if (req.Offset + req.Limit) < results.Total {
				return req.Offset + req.Limit
			}
			return req.Offset
		}(),
	}

	// Log the search request for analytics
	h.log.Info().
		Str("query", req.Query).
		Strs("types", req.Types).
		Int("limit", req.Limit).
		Int("offset", req.Offset).
		Int("resultCount", results.Total).
		Dur("duration", time.Since(start)).
		Str("orgID", reqCtx.OrgID.String()).
		Str("buID", reqCtx.BuID.String()).
		Msg("search executed")

	return c.JSON(resp)
}

// AdvancedSearch handles POST requests for more complex search operations
func (h *Handler) AdvancedSearch(c *fiber.Ctx) error {
	start := time.Now()

	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse request body
	var req struct {
		Query     string         `json:"query" validate:"required"`
		Types     []string       `json:"types,omitempty"`
		Limit     int            `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
		Offset    int            `json:"offset,omitempty" validate:"omitempty,min=0"`
		Filter    string         `json:"filter,omitempty"`
		Facets    []string       `json:"facets,omitempty"`
		SortBy    []string       `json:"sortBy,omitempty"`
		Fields    []string       `json:"fields,omitempty"`
		Highlight bool           `json:"highlight,omitempty"`
		Options   map[string]any `json:"options,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.log.Error().Err(err).Msg("failed to parse advanced search request body")
		return h.eh.HandleError(c, errors.NewBusinessError("invalid request body"))
	}

	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = 20
	}

	// Execute search with advanced options
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Facets:    req.Facets,
		SortBy:    req.SortBy,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		h.log.Error().Err(err).Interface("request", req).Msg("advanced search execution failed")
		return h.eh.HandleError(c, err)
	}

	// Build advanced response
	resp := Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
		Facets:      results.Facets,
		Metadata: map[string]any{
			"hasMore": (req.Offset + req.Limit) < results.Total,
			"nextOffset": func() int {
				if (req.Offset + req.Limit) < results.Total {
					return req.Offset + req.Limit
				}
				return req.Offset
			}(),
		},
	}

	h.log.Info().
		Str("query", req.Query).
		Interface("types", req.Types).
		Int("resultCount", results.Total).
		Dur("duration", time.Since(start)).
		Msg("advanced search executed")

	return c.Status(http.StatusOK).JSON(resp)
}

// SearchShipments provides a specialized endpoint for searching shipments
func (h *Handler) SearchShipments(c *fiber.Ctx) error {
	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse query parameters
	var req Request
	if err = c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, errors.NewValidationError("query", errors.ErrInvalid, "invalid query parameters"))
	}

	// Force shipment type
	req.Types = []string{"shipment"}

	// Execute search
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
		Facets:      results.Facets,
	})
}

// SearchEquipment provides a specialized endpoint for searching equipment
func (h *Handler) SearchEquipment(c *fiber.Ctx) error {
	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse query parameters
	var req Request
	if err = c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, errors.NewValidationError("query", errors.ErrInvalid, "invalid query parameters"))
	}

	// Force equipment type
	req.Types = []string{"equipment"}

	// Execute search
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
	})
}

// SearchCustomers provides a specialized endpoint for searching customers
func (h *Handler) SearchCustomers(c *fiber.Ctx) error {
	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse query parameters
	var req Request
	if err = c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, errors.NewValidationError("query", errors.ErrInvalid, "invalid query parameters"))
	}

	// Force customer type
	req.Types = []string{"customer"}

	// Execute search
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
	})
}

// SearchDrivers provides a specialized endpoint for searching drivers
func (h *Handler) SearchDrivers(c *fiber.Ctx) error {
	// Get request context
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse query parameters
	var req Request
	if err = c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, errors.NewValidationError("query", errors.ErrInvalid, "invalid query parameters"))
	}

	// Force driver type
	req.Types = []string{"driver"}

	// Execute search
	searchReq := &search.SearchRequest{
		Query:     req.Query,
		Types:     req.Types,
		Limit:     req.Limit,
		Offset:    req.Offset,
		OrgID:     reqCtx.OrgID.String(),
		BuID:      reqCtx.BuID.String(),
		Filter:    req.Filter,
		Highlight: req.Highlight,
	}

	results, err := h.ss.Search(c.UserContext(), searchReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(Response{
		Results:     results.Results,
		Total:       results.Total,
		ProcessedIn: fmt.Sprintf("%dms", results.ProcessedIn.Milliseconds()),
		Query:       req.Query,
		Offset:      req.Offset,
		Limit:       req.Limit,
	})
}

// Suggest provides autocomplete suggestions for partial queries
func (h *Handler) Suggest(c *fiber.Ctx) error {
	// Parse query parameters
	q := c.Query("q")
	if q == "" {
		return h.eh.HandleError(c, errors.NewValidationError("q", errors.ErrRequired, "query is required"))
	}

	types := c.Query("types")
	typesList := []string{}
	if types != "" {
		typesList = strings.Split(types, ",")
	}

	limit := 10 // Default limit for suggestions
	if c.Query("limit") != "" {
		limitVal, err := strconv.Atoi(c.Query("limit"))
		if err == nil && limitVal > 0 {
			limit = limitVal
			if limit > 50 {
				limit = 50 // Cap suggestions at 50
			}
		}
	}

	// Get suggestions from service
	suggestions, err := h.ss.GetSuggestions(c.UserContext(), q, limit, typesList)
	if err != nil {
		h.log.Error().Err(err).Str("prefix", q).Msg("suggestion generation failed")
		return h.eh.HandleError(c, err)
	}

	// Format for autocomplete use
	result := make([]fiber.Map, 0, len(suggestions))
	for i, suggestion := range suggestions {
		result = append(result, fiber.Map{
			"id":    i,
			"text":  suggestion,
			"value": suggestion,
		})
	}

	return c.JSON(fiber.Map{
		"suggestions": result,
		"query":       q,
		"total":       len(result),
	})
}

// GetHealth provides information about the search service health
func (h *Handler) GetHealth(c *fiber.Ctx) error {
	health, err := h.ss.GetHealth(c.UserContext())
	if err != nil {
		h.log.Error().Err(err).Msg("health check failed")
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"healthy": false,
			"error":   err.Error(),
		})
	}

	if healthy, ok := health["healthy"].(bool); ok && !healthy {
		return c.Status(http.StatusServiceUnavailable).JSON(health)
	}

	return c.JSON(health)
}
