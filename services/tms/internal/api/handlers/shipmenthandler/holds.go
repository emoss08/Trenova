package shipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	_ "github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// @Summary List shipment holds
// @ID listShipmentHolds
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]shipment.ShipmentHold]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/holds/ [get]
func (h *Handler) listHolds(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipment.ShipmentHold], error) {
			return h.holdService.ListByShipmentID(
				c.Request.Context(),
				&repositories.ListShipmentHoldsRequest{
					Filter:     req,
					ShipmentID: shipmentID,
				},
			)
		},
	)
}

// @Summary Get a shipment hold
// @ID getShipmentHold
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param holdID path string true "Hold ID"
// @Success 200 {object} shipment.ShipmentHold
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/holds/{holdID}/ [get]
func (h *Handler) getHold(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	holdID, err := pulid.MustParse(c.Param("holdID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.holdService.GetByID(
		c.Request.Context(),
		&repositories.GetShipmentHoldByIDRequest{
			HoldID:     holdID,
			ShipmentID: shipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a shipment hold
// @ID createShipmentHold
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body repositories.CreateShipmentHoldRequest true "Shipment hold payload"
// @Success 201 {object} shipment.ShipmentHold
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/holds/ [post]
func (h *Handler) createHold(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(repositories.CreateShipmentHoldRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ShipmentID = shipmentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	created, err := h.holdService.Create(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a shipment hold
// @ID updateShipmentHold
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param holdID path string true "Hold ID"
// @Param request body repositories.UpdateShipmentHoldRequest true "Shipment hold payload"
// @Success 200 {object} shipment.ShipmentHold
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/holds/{holdID}/ [put]
func (h *Handler) updateHold(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	holdID, err := pulid.MustParse(c.Param("holdID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(repositories.UpdateShipmentHoldRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.HoldID = holdID
	req.ShipmentID = shipmentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	updated, err := h.holdService.Update(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Release a shipment hold
// @ID releaseShipmentHold
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param holdID path string true "Hold ID"
// @Success 200 {object} shipment.ShipmentHold
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/holds/{holdID}/release/ [post]
func (h *Handler) releaseHold(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	holdID, err := pulid.MustParse(c.Param("holdID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	released, err := h.holdService.Release(
		c.Request.Context(),
		&repositories.ReleaseShipmentHoldRequest{
			HoldID:     holdID,
			ShipmentID: shipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, released)
}
