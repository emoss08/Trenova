package integration

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/googlemaps"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	IntegrationService services.IntegrationService
	GoogleMapsClient   googlemaps.Client
	ErrorHandler       *validator.ErrorHandler
}

type Handler struct {
	integrationService services.IntegrationService
	googleMapsClient   googlemaps.Client
	errorHandler       *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		integrationService: p.IntegrationService,
		googleMapsClient:   p.GoogleMapsClient,
		errorHandler:       p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router, rl *middleware.RateLimiter) {
	api := router.Group("/integrations")
	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(60),
	)...)
	api.Post("/google-maps/autocomplete", rl.WithRateLimit(
		[]fiber.Handler{h.googleAutocomplete},
		middleware.PerMinute(60),
	)...)
	api.Get("/:type/", rl.WithRateLimit(
		[]fiber.Handler{h.getByType},
		middleware.PerMinute(60),
	)...)
	api.Get("/:id/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60),
	)...)
	api.Put("/:id/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(30),
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*integration.Integration], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.errorHandler.HandleError(fc, err)
		}

		return h.integrationService.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.errorHandler, reqCtx, handler)
}

func (h *Handler) getByType(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	result, err := h.integrationService.GetByType(c.UserContext(), repositories.GetIntegrationByTypeRequest{
		Type:  integration.Type(c.Params("type")),
		OrgID: reqCtx.OrgID,
		BuID:  reqCtx.BuID,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	integrationID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}
	result, err := h.integrationService.GetByID(c.UserContext(), repositories.GetIntegrationByIDOptions{
		ID:     integrationID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// Update updates an integration
func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	integrationID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	intgr := new(integration.Integration)
	intgr.ID = integrationID
	intgr.OrganizationID = reqCtx.OrgID
	intgr.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(intgr); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	result, err := h.integrationService.Update(c.UserContext(), intgr, reqCtx.UserID)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) googleAutocomplete(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	req := new(googlemaps.AutoCompleteRequest)
	if err = c.BodyParser(req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// * Set the org and bu ids
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID

	resp, err := h.googleMapsClient.AutocompleteWithDetails(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
