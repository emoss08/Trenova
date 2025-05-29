package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/api/middleware"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipment"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/utils/streamingutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	ShipmentService *shipment.Service
	ErrorHandler    *validator.ErrorHandler
}

type Handler struct {
	ss *shipment.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.ShipmentService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipments")
	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/live", h.liveStream)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/cancel/", rl.WithRateLimit(
		[]fiber.Handler{h.cancel},
		middleware.PerSecond(5), // 5 writes per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/:shipmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:shipmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Put("/:shipmentID/mark-ready-to-bill/", rl.WithRateLimit(
		[]fiber.Handler{h.markReadyToBill},
		middleware.PerMinute(5), // 5 writes per minute
	)...)

	api.Post("/duplicate/", rl.WithRateLimit(
		[]fiber.Handler{h.duplicate},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Post("/calculate-totals/", rl.WithRateLimit(
		[]fiber.Handler{h.calculateTotals},
		middleware.PerSecond(30), // allow a few per second for debounced UI calls
	)...)

	api.Post("/check-for-duplicate-bols/", rl.WithRateLimit(
		[]fiber.Handler{h.checkForDuplicateBOLs},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListShipmentOptions{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  100,
			Offset: 0,
		},
	}

	options, err := h.ss.SelectOptions(c.UserContext(), opts)
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

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*shipmentdomain.Shipment], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ss.List(fc.UserContext(), &repositories.ListShipmentOptions{
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: c.QueryBool("expandShipmentDetails"),
				Status:                c.Query("status"),
			},
			Filter: filter,
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shipmentID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp, err := h.ss.Get(c.UserContext(), &repositories.GetShipmentByIDOptions{
		ID:     shipmentID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: c.QueryBool("expandShipmentDetails", false),
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(shp)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp := new(shipmentdomain.Shipment)
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ss.Create(c.UserContext(), shp, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shpID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp := new(shipmentdomain.Shipment)
	shp.ID = shpID
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ss.Update(c.UserContext(), shp, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) cancel(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.CancelShipmentRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	newEntity, err := h.ss.Cancel(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newEntity)
}

func (h *Handler) duplicate(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.DuplicateShipmentRequest)
	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID
	req.UserID = reqCtx.UserID

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	newEntity, err := h.ss.Duplicate(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newEntity)
}

func (h *Handler) markReadyToBill(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.UpdateShipmentStatusRequest)
	shipmentID, err := pulid.MustParse(c.Params("shipmentID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req.GetOpts.ID = shipmentID
	req.GetOpts.BuID = reqCtx.BuID
	req.GetOpts.OrgID = reqCtx.OrgID
	req.GetOpts.UserID = reqCtx.UserID

	updatedEntity, err := h.ss.MarkReadyToBill(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}

// BOLCheckRequest represents the request structure for BOL duplicate checking
type BOLCheckRequest struct {
	BOL        string    `json:"bol"`
	ShipmentID *pulid.ID `json:"shipmentId,omitempty"` // Optional, for excluding current shipment during updates
}

// BOLCheckResponse represents the response structure for the BOL check endpoint
type BOLCheckResponse struct {
	Valid bool `json:"valid"`
}

func (h *Handler) checkForDuplicateBOLs(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// Parse request
	req := new(BOLCheckRequest)
	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Skip check if BOL is empty
	if req.BOL == "" {
		return c.Status(fiber.StatusOK).JSON(BOLCheckResponse{
			Valid: true,
		})
	}

	// Create shipment with required data for the check
	shp := new(shipmentdomain.Shipment)
	shp.BOL = req.BOL
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	// Set ID if provided (for excluding current shipment during updates)
	if req.ShipmentID != nil && !req.ShipmentID.IsNil() {
		shp.ID = *req.ShipmentID
	}

	// Check for duplicates
	if err = h.ss.CheckForDuplicateBOLs(c.UserContext(), shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	// If no errors, the BOL is valid
	return c.Status(fiber.StatusOK).JSON(BOLCheckResponse{
		Valid: true,
	})
}

// calculateTotals provides a stateless endpoint that receives a (partial)
// shipment payload and returns the calculated monetary totals. It never
// persists data – it merely reuses the same calculator that runs during
// create/update operations.
func (h *Handler) calculateTotals(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	shp := new(shipmentdomain.Shipment)
	shp.OrganizationID = reqCtx.OrgID
	shp.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(shp); err != nil {
		return h.eh.HandleError(c, err)
	}

	resp, err := h.ss.CalculateShipmentTotals(shp)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) liveStream(c *fiber.Ctx) error {
	// Use the simplified streaming helper for shipments
	fetchFunc := func(ctx context.Context, reqCtx *appctx.RequestContext) ([]*shipmentdomain.Shipment, error) {
		filter := &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				BuID:   reqCtx.BuID,
				OrgID:  reqCtx.OrgID,
				UserID: reqCtx.UserID,
			},
			Limit:  10, // Get last 10 shipments
			Offset: 0,
		}

		result, err := h.ss.List(ctx, &repositories.ListShipmentOptions{
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: false, // Keep it lightweight for streaming
			},
			Filter: filter,
		})
		if err != nil {
			return nil, err
		}

		return result.Items, nil
	}

	timestampFunc := func(shipment *shipmentdomain.Shipment) int64 {
		// Use CreatedAt to only track new shipments, not existing ones
		return shipment.CreatedAt
	}

	return streamingutils.StreamWithSimplePoller(
		c,
		streamingutils.DefaultSSEConfig(),
		fetchFunc,
		timestampFunc,
	)
}
