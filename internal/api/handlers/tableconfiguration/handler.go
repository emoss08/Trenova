package tableconfiguration

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	tableconfigurationdomain "github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tableconfiguration"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ts *tableconfiguration.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	TableConfigurationService *tableconfiguration.Service
	ErrorHandler              *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ts: p.TableConfigurationService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/table-configurations")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60),
	)...)

	// Retrieve or create configuration for current user by table identifier
	api.Get(":tableIdentifier", rl.WithRateLimit(
		[]fiber.Handler{h.getDefaultOrLatestConfiguration},
		middleware.PerMinute(60),
	)...)

	// Partial update of configuration JSON blob
	api.Patch(":configID", rl.WithRateLimit(
		[]fiber.Handler{h.patch},
		middleware.PerMinute(60),
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.FilterQueryOptions{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
		Query:  c.Query("query"),
	}

	result, err := h.ts.List(c.UserContext(),
		&repositories.TableConfigurationFilters{
			Base:           opts,
			Search:         c.Query("search"),
			IncludeShares:  c.Query("include_shares") == "true",
			IncludeCreator: c.Query("include_creator") == "true",
		})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	config := new(tableconfigurationdomain.Configuration)
	config.OrganizationID = reqCtx.OrgID
	config.BusinessUnitID = reqCtx.BuID
	config.UserID = reqCtx.UserID

	if err = c.BodyParser(config); err != nil {
		return h.eh.HandleError(c, err)
	}

	created, err := h.ts.Create(c.UserContext(), config)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(created)
}

// getDefaultOrLatestConfiguration returns a configuration for the given tableIdentifier
// If none exists for the requesting user + org + bu, it will create a default one.
func (h *Handler) getDefaultOrLatestConfiguration(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tableID := c.Params("tableIdentifier")

	config, err := h.ts.GetDefaultOrLatestConfiguration(c.UserContext(), tableID, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	if config == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}
	return c.Status(fiber.StatusOK).JSON(config)
}

// patch allows partial updates to the tableConfig JSON blob.
func (h *Handler) patch(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var payload struct {
		TableConfig map[string]any `json:"tableConfig"`
	}

	if err = c.BodyParser(&payload); err != nil {
		return h.eh.HandleError(c, err)
	}

	configID := c.Params("configID")

	updated, err := h.ts.Patch(c.UserContext(), configID, payload.TableConfig, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updated)
}
