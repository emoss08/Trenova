package shipmentmovehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              shipmentmoveports.ShipmentMoveService
	TrackingEvents       trackingeventports.TrackingEventService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service        shipmentmoveports.ShipmentMoveService
	trackingEvents trackingeventports.TrackingEventService
	eh             *helpers.ErrorHandler
	pm             *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:        p.Service,
		trackingEvents: p.TrackingEvents,
		eh:             p.ErrorHandler,
		pm:             p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	moves := rg.Group("/shipment-moves")
	moves.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)
	moves.POST(
		"/:moveID/update-status/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUpdate),
		h.updateStatus,
	)
	moves.POST(
		"/:moveID/split/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUpdate),
		h.splitMove,
	)
	moves.POST(
		"/:moveID/stops/:stopID/record-actual/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUpdate),
		h.recordStopActual,
	)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type bulkUpdateStatusRequest struct {
	MoveIDs []pulid.ID `json:"moveIds"`
	Status  string     `json:"status"`
}

type recordStopActualRequest struct {
	Action     string `json:"action"`
	OccurredAt *int64 `json:"occurredAt,omitempty"`
}

type splitMoveRequest struct {
	NewDeliveryLocationID pulid.ID                         `json:"newDeliveryLocationId"`
	SplitPickupTimes      shipmentmoveports.SplitStopTimes `json:"splitPickupTimes"`
	NewDeliveryTimes      shipmentmoveports.SplitStopTimes `json:"newDeliveryTimes"`
	Pieces                *int64                           `json:"pieces,omitempty"`
	Weight                *int64                           `json:"weight,omitempty"`
}

// @Summary Update a shipment move status
// @ID updateShipmentMoveStatus
// @Tags Shipment Moves
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param request body updateStatusRequest true "Status update request"
// @Success 200 {object} shipment.ShipmentMove
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/update-status/ [post]
func (h *Handler) updateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(updateStatusRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.UpdateStatus(
		c.Request.Context(),
		&shipmentmoveports.UpdateMoveStatusRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			MoveID:     moveID,
			Status:     shipmentvocab.MoveStatus(body.Status),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Bulk update shipment move statuses
// @ID bulkUpdateShipmentMoveStatus
// @Tags Shipment Moves
// @Accept json
// @Produce json
// @Param request body bulkUpdateStatusRequest true "Bulk status update request"
// @Success 200 {array} shipment.ShipmentMove
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	body := new(bulkUpdateStatusRequest)
	if err := c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entities, err := h.service.BulkUpdateStatus(
		c.Request.Context(),
		&shipmentmoveports.BulkUpdateMoveStatusRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			MoveIDs:    body.MoveIDs,
			Status:     shipmentvocab.MoveStatus(body.Status),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entities)
}

// @Summary Record a stop actual arrival or departure on a shipment move
// @ID recordShipmentMoveStopActual
// @Tags Shipment Moves
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param stopID path string true "Stop ID"
// @Param request body recordStopActualRequest true "Stop actual request"
// @Success 200 {object} shipment.ShipmentMove
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/stops/{stopID}/record-actual/ [post]
func (h *Handler) recordStopActual(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	stopID, err := pulid.MustParse(c.Param("stopID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(recordStopActualRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	ctx := c.Request.Context()
	sourceKey, _ := idempotency.KeyFrom(ctx)
	result, err := h.trackingEvents.RecordReported(
		ctx,
		&trackingeventports.RecordReportedStopEventParams{
			TenantInfo:   pagination.FromAuthAsUser(authCtx),
			MoveID:       moveID,
			StopID:       stopID,
			Action:       shipmentmoveports.StopActualAction(body.Action),
			OccurredAt:   body.OccurredAt,
			Source:       trackingevent.SourceDispatcher,
			SourceKey:    sourceKey,
			ReportedByID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result.Move)
}

// @Summary Split a shipment move
// @ID splitShipmentMove
// @Tags Shipment Moves
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param request body splitMoveRequest true "Split move request"
// @Success 200 {object} repositories.SplitMoveResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/split/ [post]
func (h *Handler) splitMove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(splitMoveRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.SplitMove(c.Request.Context(), &shipmentmoveports.SplitMoveRequest{
		TenantInfo:            pagination.FromAuth(authCtx),
		MoveID:                moveID,
		NewDeliveryLocationID: body.NewDeliveryLocationID,
		SplitPickupTimes:      body.SplitPickupTimes,
		NewDeliveryTimes:      body.NewDeliveryTimes,
		Pieces:                body.Pieces,
		Weight:                body.Weight,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}
