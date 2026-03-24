package shipmentmovehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              portservices.ShipmentMoveService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service portservices.ShipmentMoveService
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
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type bulkUpdateStatusRequest struct {
	MoveIDs []pulid.ID `json:"moveIds"`
	Status  string     `json:"status"`
}

type splitMoveRequest struct {
	NewDeliveryLocationID pulid.ID                    `json:"newDeliveryLocationId"`
	SplitPickupTimes      repositories.SplitStopTimes `json:"splitPickupTimes"`
	NewDeliveryTimes      repositories.SplitStopTimes `json:"newDeliveryTimes"`
	Pieces                *int64                      `json:"pieces,omitempty"`
	Weight                *int64                      `json:"weight,omitempty"`
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
		&repositories.UpdateMoveStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			MoveID: moveID,
			Status: shipment.MoveStatus(body.Status),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
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
		&repositories.BulkUpdateMoveStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			MoveIDs: body.MoveIDs,
			Status:  shipment.MoveStatus(body.Status),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
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

	entity, err := h.service.SplitMove(c.Request.Context(), &repositories.SplitMoveRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
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

	c.JSON(http.StatusOK, entity)
}
