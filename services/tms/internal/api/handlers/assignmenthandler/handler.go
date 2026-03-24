package assignmenthandler

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

	Service              portservices.AssignmentService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service portservices.AssignmentService
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
	assignments := rg.Group("/assignments")
	assignments.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpRead),
		h.list,
	)
	assignments.GET(
		"/:assignmentID/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpRead),
		h.get,
	)

	moveAssignments := rg.Group("/shipment-moves/:moveID/assignment")
	moveAssignments.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpAssign),
		h.assignToMove,
	)
	moveAssignments.PUT(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpAssign),
		h.reassign,
	)
	moveAssignments.DELETE(
		"/",
		h.pm.RequirePermission(permission.ResourceShipmentMove.String(), permission.OpUnassign),
		h.unassign,
	)
}

// @Summary List assignments
// @ID listAssignments
// @Tags Assignments
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]shipment.Assignment]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /assignments/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipment.Assignment], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAssignmentsRequest{
				Filter: req,
			})
		},
	)
}

// @Summary Get an assignment
// @ID getAssignment
// @Tags Assignments
// @Produce json
// @Param assignmentID path string true "Assignment ID"
// @Success 200 {object} shipment.Assignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /assignments/{assignmentID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	assignmentID, err := pulid.MustParse(c.Param("assignmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetAssignmentByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			AssignmentID: assignmentID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type upsertAssignmentRequest struct {
	PrimaryWorkerID   pulid.ID  `json:"primaryWorkerId"`
	TractorID         pulid.ID  `json:"tractorId"`
	TrailerID         *pulid.ID `json:"trailerId,omitempty"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId,omitempty"`
}

// @Summary Assign a move
// @ID assignShipmentMove
// @Tags Assignments
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param request body upsertAssignmentRequest true "Assignment payload"
// @Success 201 {object} shipment.Assignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/assignment/ [post]
func (h *Handler) assignToMove(c *gin.Context) {
	h.upsert(c, true)
}

// @Summary Reassign a move
// @ID reassignShipmentMove
// @Tags Assignments
// @Accept json
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Param request body upsertAssignmentRequest true "Assignment payload"
// @Success 200 {object} shipment.Assignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/assignment/ [put]
func (h *Handler) reassign(c *gin.Context) {
	h.upsert(c, false)
}

// @Summary Unassign a move
// @ID unassignShipmentMove
// @Tags Assignments
// @Param moveID path string true "Shipment move ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/assignment/ [delete]
func (h *Handler) unassign(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Unassign(
		c.Request.Context(),
		&repositories.UnassignShipmentMoveRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			ShipmentMoveID: moveID,
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) upsert(c *gin.Context, create bool) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(upsertAssignmentRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	if create {
		entity, svcErr := h.service.AssignToMove(
			c.Request.Context(),
			&repositories.AssignShipmentMoveRequest{
				TenantInfo:        tenantInfo,
				ShipmentMoveID:    moveID,
				PrimaryWorkerID:   req.PrimaryWorkerID,
				TractorID:         req.TractorID,
				TrailerID:         req.TrailerID,
				SecondaryWorkerID: req.SecondaryWorkerID,
			},
		)
		if svcErr != nil {
			h.eh.HandleError(c, svcErr)
			return
		}

		c.JSON(http.StatusCreated, entity)
		return
	}

	entity, svcErr := h.service.Reassign(
		c.Request.Context(),
		&repositories.ReassignShipmentMoveRequest{
			TenantInfo:        tenantInfo,
			ShipmentMoveID:    moveID,
			PrimaryWorkerID:   req.PrimaryWorkerID,
			TractorID:         req.TractorID,
			TrailerID:         req.TrailerID,
			SecondaryWorkerID: req.SecondaryWorkerID,
		},
	)
	if svcErr != nil {
		h.eh.HandleError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, entity)
}
