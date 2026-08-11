package carrierassignmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *carrierassignmentservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *carrierassignmentservice.Service
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
	moveAssignments := rg.Group("/shipment-moves/:moveID/carrier-assignment")
	moveAssignments.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpRead),
		h.get,
	)
	moveAssignments.GET(
		"/preview/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpRead),
		h.preview,
	)
	moveAssignments.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpAssign),
		h.assign,
	)
	moveAssignments.DELETE(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUnassign),
		h.cancel,
	)
}

// @Summary Get the active carrier assignment for a move
// @ID getMoveCarrierAssignment
// @Tags CarrierAssignments
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Success 200 {object} shipment.CarrierAssignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/carrier-assignment/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetActiveByMoveID(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		moveID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if entity == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Preview carrier eligibility for a move assignment
// @ID previewMoveCarrierAssignment
// @Tags CarrierAssignments
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param carrierId query string true "Carrier ID"
// @Success 200 {object} carrier.EligibilityResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/carrier-assignment/preview/ [get]
func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	carrierID, err := pulid.MustParse(c.Query("carrierId"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.PreviewEligibility(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		carrierID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Assign a move to an external carrier
// @ID assignMoveToCarrier
// @Tags CarrierAssignments
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param request body repositories.AssignMoveToCarrierRequest true "Carrier assignment payload"
// @Success 201 {object} shipment.CarrierAssignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/carrier-assignment/ [post]
func (h *Handler) assign(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(repositories.AssignMoveToCarrierRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.ShipmentMoveID = moveID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	entity, err := h.service.AssignToMove(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

type cancelCarrierAssignmentBody struct {
	Reason string `json:"reason"`
}

// @Summary Cancel the carrier assignment for a move
// @ID cancelMoveCarrierAssignment
// @Tags CarrierAssignments
// @Accept json
// @Param moveID path string true "Shipment move ID"
// @Param request body cancelCarrierAssignmentBody true "Cancellation payload"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/carrier-assignment/ [delete]
func (h *Handler) cancel(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(cancelCarrierAssignmentBody)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Cancel(
		c.Request.Context(),
		&repositories.CancelCarrierAssignmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			ShipmentMoveID: moveID,
			Reason:         body.Reason,
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
