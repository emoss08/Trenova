package resourceeditor

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ResourceEditorRepository repositories.ResourceEditorRepository
	ErrorHandler             *validator.ErrorHandler
}

type Handler struct {
	resourceEditorRepository repositories.ResourceEditorRepository
	eh                       *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		resourceEditorRepository: p.ResourceEditorRepository,
		eh:                       p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/resource-editor")

	api.Get("/table-schema/", rl.WithRateLimit(
		[]fiber.Handler{h.GetTableSchema},
		middleware.PerMinute(120),
	)...)

	// New route for autocomplete suggestions
	api.Post("/autocomplete/", rl.WithRateLimit(
		[]fiber.Handler{h.GetAutocompleteSuggestionsHandler},
		middleware.PerMinute(300), // Allow more frequent calls for autocomplete
	)...)

	// New route for SQL query execution
	api.Post("/execute-query/", rl.WithRateLimit(
		[]fiber.Handler{h.ExecuteQueryHandler},
		middleware.PerMinute(60), // Rate limit query execution
	)...)
}

func (h *Handler) GetTableSchema(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	schema, err := h.resourceEditorRepository.GetTableSchema(c.UserContext(), "")
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(schema)
}

// GetAutocompleteSuggestionsHandler handles requests for SQL autocompletion suggestions.
func (h *Handler) GetAutocompleteSuggestionsHandler(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.AutocompleteRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, eris.Wrap(err, "failed to parse autocomplete request body"))
	}

	// You might want to pull the schema name from a session or default it if not provided in req
	if req.SchemaName == "" {
		req.SchemaName = "public" // Default to public schema
	}

	suggestions, err := h.resourceEditorRepository.GetAutocompleteSuggestions(c.UserContext(), req)
	if err != nil {
		// The repository method already logs, so just return the error
		return h.eh.HandleError(c, err) // eris.Wrap(err, "failed to get autocomplete suggestions") is handled by eh
	}

	return c.Status(fiber.StatusOK).JSON(suggestions)
}

// ExecuteQueryHandler handles requests for executing SQL queries.
func (h *Handler) ExecuteQueryHandler(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.ExecuteQueryRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, eris.Wrap(err, "failed to parse execute query request body"))
	}

	// Ensure query is not empty
	if req.Query == "" {
		return h.eh.HandleError(c, eris.New("SQL query cannot be empty"))
	}

	// Default schema if not provided, though the query itself should be self-contained or use this as context
	if req.SchemaName == "" {
		req.SchemaName = "public"
	}

	resp, err := h.resourceEditorRepository.ExecuteSQLQuery(c.UserContext(), req)
	if err != nil {
		// This error is from the repository function itself (e.g., DB connection failed before trying query)
		return h.eh.HandleError(c, err)
	}

	// If resp.Result.Error is populated, it means the query execution had an issue.
	// The HTTP status might still be 200 OK here, as the API call itself succeeded.
	// The client will inspect the `result.error` field.
	return c.Status(fiber.StatusOK).JSON(resp)
}
