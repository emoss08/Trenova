// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package consolidation

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	consolidationservice "github.com/emoss08/trenova/internal/core/services/consolidation"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// HandlerParams defines dependencies required for initializing the ConsolidationHandler.
type HandlerParams struct {
	fx.In

	ConsolidationService *consolidationservice.Service
	ErrorHandler         *validator.ErrorHandler
}

// Handler handles HTTP requests for consolidation operations.
type Handler struct {
	cs *consolidationservice.Service
	eh *validator.ErrorHandler
}

// NewHandler creates a new consolidation handler instance.
//
// Parameters:
//   - p: HandlerParams containing required dependencies.
//
// Returns:
//   - *Handler: A ready-to-use consolidation handler instance.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{cs: p.ConsolidationService, eh: p.ErrorHandler}
}

// RegisterRoutes registers all consolidation-related routes.
//
// Parameters:
//   - r: Fiber router to register routes on.
//   - rl: Rate limiter middleware for rate limiting endpoints.
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/consolidations")

	// * List endpoints
	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	// * Create endpoint
	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 creates per minute
	)...)

	// * Single consolidation endpoints
	api.Get("/:consolidationID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:consolidationID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 updates per minute
	)...)

	api.Post("/:consolidationID/cancel/", rl.WithRateLimit(
		[]fiber.Handler{h.cancel},
		middleware.PerMinute(10), // 10 cancellations per minute
	)...)

	// * Shipment management endpoints
	api.Post("/:consolidationID/shipments/:shipmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.addShipment},
		middleware.PerMinute(60), // 60 additions per minute
	)...)

	api.Delete("/:consolidationID/shipments/:shipmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.removeShipment},
		middleware.PerMinute(60), // 60 removals per minute
	)...)

	api.Get("/:consolidationID/shipments/", rl.WithRateLimit(
		[]fiber.Handler{h.getShipments},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

// selectOptions returns consolidation groups as select options for UI dropdowns.
func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.QueryOptions{
		TenantOpts: ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  100,
		Offset: 0,
	}

	options, err := h.cs.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Count:   len(options),
		Next:    "",
		Prev:    "",
	})
}

// list retrieves paginated consolidation groups.
func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	eo, err := paginationutils.ParseEnhancedQueryFromJSON(c, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	qo := new(repositories.ConsolidationOptions)
	if err = paginationutils.ParseAdditionalQueryParams(c, qo); err != nil {
		return h.eh.HandleError(c, err)
	}

	listOpts := repositories.BuildConsolidationListOptions(eo, *qo)

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*consolidation.ConsolidationGroup], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.cs.List(fc.UserContext(), listOpts)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

// get retrieves a single consolidation group by ID.
func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	cg, err := h.cs.Get(
		c.UserContext(),
		consolidationID,
		reqCtx.UserID,
		reqCtx.OrgID,
		reqCtx.BuID,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(cg)
}

// create creates a new consolidation group.
func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(consolidation.ConsolidationGroup)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.OrganizationID = reqCtx.OrgID
	req.BusinessUnitID = reqCtx.BuID

	entity, err := h.cs.Create(c.UserContext(), req, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(entity)
}

// update updates an existing consolidation group.
func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	cg := new(consolidation.ConsolidationGroup)
	cg.ID = consolidationID
	cg.OrganizationID = reqCtx.OrgID
	cg.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(cg); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.cs.Update(c.UserContext(), cg, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

// CancelRequest represents the request body for canceling a consolidation.
type CancelRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// cancel cancels a consolidation group and all associated shipments.
func (h *Handler) cancel(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(CancelRequest)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.cs.CancelConsolidation(
		c.UserContext(),
		consolidationID,
		reqCtx.UserID,
		reqCtx.OrgID,
		reqCtx.BuID,
		req.Reason,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Consolidation group canceled successfully",
	})
}

// addShipment adds a shipment to a consolidation group.
func (h *Handler) addShipment(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.cs.AddShipmentToGroup(
		c.UserContext(),
		consolidationID,
		shipmentID,
		reqCtx.UserID,
		reqCtx.OrgID,
		reqCtx.BuID,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Shipment added to consolidation group successfully",
	})
}

// removeShipment removes a shipment from a consolidation group.
func (h *Handler) removeShipment(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.cs.RemoveShipmentFromGroup(
		c.UserContext(),
		consolidationID,
		shipmentID,
		reqCtx.UserID,
		reqCtx.OrgID,
		reqCtx.BuID,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Shipment removed from consolidation group successfully",
	})
}

// getShipments retrieves all shipments in a consolidation group.
func (h *Handler) getShipments(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	consolidationID, err := pulid.MustParse(c.Params("consolidationID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipments, err := h.cs.GetGroupShipments(
		c.UserContext(),
		consolidationID,
		reqCtx.UserID,
		reqCtx.OrgID,
		reqCtx.BuID,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(shipments)
}
