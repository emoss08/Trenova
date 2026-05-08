package shipmenteventhandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	defaultEventLimit = 10
	maxEventLimit     = 100
)

type Params struct {
	fx.In

	Service              services.ShipmentEventService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.ShipmentEventService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipment-events")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.list,
	)
}

// @Summary List shipment events
// @ID listShipmentEvents
// @Tags Shipment Events
// @Produce json
// @Param shipmentId query string false "Filter to a single shipment"
// @Param types query string false "Comma-separated event types"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param before query int false "Cursor: occurred_at, exclusive"
// @Success 200 {array} shipmentevent.Event
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-events/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := &repositories.ListShipmentEventsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Limit: parseLimit(c.Query("limit")),
	}

	if raw := strings.TrimSpace(c.Query("shipmentId")); raw != "" {
		shipmentID, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		req.ShipmentID = shipmentID
	}

	if raw := strings.TrimSpace(c.Query("types")); raw != "" {
		req.Types = parseTypes(raw)
	}

	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		before, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		req.Before = before
	}

	events, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

func parseLimit(raw string) int {
	if raw == "" {
		return defaultEventLimit
	}
	limit, err := strconv.Atoi(raw)
	switch {
	case err != nil, limit <= 0:
		return defaultEventLimit
	case limit > maxEventLimit:
		return maxEventLimit
	default:
		return limit
	}
}

func parseTypes(raw string) []shipmentevent.Type {
	parts := strings.Split(raw, ",")
	types := make([]shipmentevent.Type, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		types = append(types, shipmentevent.Type(trimmed))
	}
	return types
}
