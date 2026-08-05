package permithandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              services.PermitService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service services.PermitService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
	logger  *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
		logger:  p.Logger.Named("api.permit-handler"),
	}
}

// RegisterRoutes mounts permits under the shipment they belong to. The wildcard
// segment must stay named :shipmentID to match the shipment handler's own
// registration — gin panics at startup on two different names for the same
// position in the route tree.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipments")

	api.GET(
		"/:shipmentID/permits/",
		h.pm.RequirePermission(permission.ResourcePermit.String(), permission.OpRead),
		h.listPermits,
	)
	api.POST(
		"/:shipmentID/permits/",
		h.pm.RequirePermission(permission.ResourcePermit.String(), permission.OpCreate),
		h.createPermit,
	)
	api.PUT(
		"/:shipmentID/permits/:permitID/",
		h.pm.RequirePermission(permission.ResourcePermit.String(), permission.OpUpdate),
		h.updatePermit,
	)
	api.GET(
		"/:shipmentID/permit-requirements/",
		h.pm.RequirePermission(permission.ResourcePermit.String(), permission.OpRead),
		h.listRequirements,
	)
	api.POST(
		"/:shipmentID/permit-requirements/:requirementID/waive/",
		h.pm.RequirePermission(permission.ResourcePermit.String(), permission.OpApprove),
		h.waiveRequirement,
	)
}

// @Summary List permits for a shipment
// @ID listShipmentPermits
// @Tags Permits
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {array} permit.Permit
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permits/ [get]
func (h *Handler) listPermits(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	permits, err := h.service.ListPermits(c.Request.Context(), shipmentID, actorutil.TenantInfoFrom(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, permits)
}

// @Summary Record a permit for a shipment
// @ID createShipmentPermit
// @Tags Permits
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body permit.Permit true "Permit payload"
// @Success 201 {object} permit.Permit
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permits/ [post]
func (h *Handler) createPermit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(permit.Permit)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// The path owns the shipment and the session owns the tenant. Taking either
	// from the body would let a caller record a permit against someone else's
	// load by editing the payload.
	entity.ShipmentID = shipmentID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	created, err := h.service.CreatePermit(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a recorded permit
// @ID updateShipmentPermit
// @Tags Permits
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param permitID path string true "Permit ID"
// @Param request body permit.Permit true "Permit payload"
// @Success 200 {object} permit.Permit
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permits/{permitID}/ [put]
func (h *Handler) updatePermit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	permitID, err := pulid.MustParse(c.Param("permitID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(permit.Permit)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = permitID
	entity.ShipmentID = shipmentID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	updated, err := h.service.UpdatePermit(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary List derived permit requirements for a shipment
// @ID listShipmentPermitRequirements
// @Tags Permits
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {array} permit.Requirement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permit-requirements/ [get]
func (h *Handler) listRequirements(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	requirements, err := h.service.ListRequirements(
		c.Request.Context(), shipmentID, actorutil.TenantInfoFrom(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, requirements)
}

type waiveRequirementBody struct {
	Reason string `json:"reason"`
}

// @Summary Waive a permit requirement
// @Description Marks a derived permit requirement as waived. This does not make the movement legal; it records that the organization accepts the compliance risk, and the reason is the audit trail.
// @ID waiveShipmentPermitRequirement
// @Tags Permits
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param requirementID path string true "Requirement ID"
// @Param request body waiveRequirementBody true "Waiver payload"
// @Success 200 {object} permit.Requirement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permit-requirements/{requirementID}/waive/ [post]
func (h *Handler) waiveRequirement(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	requirementID, err := pulid.MustParse(c.Param("requirementID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(waiveRequirementBody)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	waived, err := h.service.WaiveRequirement(c.Request.Context(), &services.WaiveRequirementRequest{
		TenantInfo:    actorutil.TenantInfoFrom(authCtx),
		RequirementID: requirementID,
		// The waiving user comes from the session, never the payload: the point
		// of the record is who accepted the risk.
		WaivedByID: authCtx.UserID,
		Reason:     body.Reason,
		Actor:      actorutil.FromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, waived)
}
