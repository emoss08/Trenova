package shipmenthold

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenthold"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	Service      *shipmenthold.Service
	ErrorHandler *validator.ErrorHandler
}

type Handler struct {
	sh *shipmenthold.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		sh: p.Service,
		eh: p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/shipment-holds")

	api.Get("/:shipmentID/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/hold/", rl.WithRateLimit(
		[]fiber.Handler{h.holdShipment},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Post("/release/", rl.WithRateLimit(
		[]fiber.Handler{h.releaseShipmentHold},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*shipment.ShipmentHold], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		shipmentID, err := pulid.MustParse(fc.Params("shipmentID"))
		if err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.sh.List(fc.UserContext(), &repositories.ListShipmentHoldOptions{
			ShipmentID: shipmentID,
			Filter:     filter,
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) holdShipment(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.HoldShipmentRequest)

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.BuID = reqCtx.BuID
	req.OrgID = reqCtx.OrgID
	req.UserID = reqCtx.UserID

	hold, err := h.sh.HoldShipment(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hold)
}

func (h *Handler) releaseShipmentHold(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	req := new(repositories.ReleaseShipmentHoldRequest)

	if err = c.BodyParser(req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.BuID = reqCtx.BuID
	req.OrgID = reqCtx.OrgID
	req.UserID = reqCtx.UserID

	hold, err := h.sh.ReleaseShipmentHold(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(hold)
}
