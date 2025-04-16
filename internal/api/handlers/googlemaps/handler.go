package googlemaps

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/googlemaps"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	GoogleMapsClient googlemaps.Client
	ErrorHandler     *validator.ErrorHandler
}

type Handler struct {
	gmapsClient  googlemaps.Client
	errorHandler *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		gmapsClient:  p.GoogleMapsClient,
		errorHandler: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/google-maps")

	api.Post("/autocomplete/", rl.WithRateLimit(
		[]fiber.Handler{h.PlaceAutocomplete},
		middleware.PerMinute(30),
	)...)

	api.Get("/check-api-key/", rl.WithRateLimit(
		[]fiber.Handler{h.CheckAPIKey},
		middleware.PerMinute(30),
	)...)
}

func (h *Handler) PlaceAutocomplete(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	req := new(googlemaps.AutoCompleteRequest)
	if err = c.BodyParser(req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	resp, err := h.gmapsClient.AutocompleteWithDetails(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) CheckAPIKey(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	valid, err := h.gmapsClient.CheckAPIKey(c.UserContext(), reqCtx.OrgID, reqCtx.BuID)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"valid": valid,
	})
}
