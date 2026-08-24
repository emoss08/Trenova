package tenderhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *tenderservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *tenderservice.Service
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
	api := rg.Group("/tenders")
	api.GET(
		"/:tenderID/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/waterfall/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpCreate),
		h.createWaterfall,
	)
	api.POST(
		"/spot/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpCreate),
		h.createSpot,
	)
	api.POST(
		"/:tenderID/cancel/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpCancel),
		h.cancel,
	)
	api.POST(
		"/offers/:offerID/respond/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpUpdate),
		h.recordResponse,
	)
	api.GET(
		"/by-shipment/:shipmentID/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpRead),
		h.listByShipment,
	)
	api.GET(
		"/by-move/:moveID/live/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpRead),
		h.getLiveByMove,
	)
}

func tenantInfoFor(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// @Summary Get a tender with its offers
// @ID getTender
// @Tags Tenders
// @Produce json
// @Param tenderID path string true "Tender ID"
// @Success 200 {object} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/{tenderID}/ [get]
func (h *Handler) get(c *gin.Context) {
	tenderID, err := pulid.MustParse(c.Param("tenderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetTenderByIDRequest{
		TenantInfo:    tenantInfoFor(c),
		TenderID:      tenderID,
		IncludeOffers: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Start a waterfall tender from a routing guide
// @ID createWaterfallTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param request body tenderservice.CreateWaterfallTenderRequest true "Waterfall tender request"
// @Success 201 {object} tenderservice.CreateWaterfallResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/waterfall/ [post]
func (h *Handler) createWaterfall(c *gin.Context) {
	req := new(tenderservice.CreateWaterfallTenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFor(c)

	entity, err := h.service.CreateWaterfall(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

// @Summary Start a spot tender with an ad-hoc carrier list
// @ID createSpotTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param request body tenderservice.CreateSpotTenderRequest true "Spot tender request"
// @Success 201 {object} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/spot/ [post]
func (h *Handler) createSpot(c *gin.Context) {
	req := new(tenderservice.CreateSpotTenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFor(c)

	entity, err := h.service.CreateSpot(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

// @Summary Cancel a live tender
// @ID cancelTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param tenderID path string true "Tender ID"
// @Param request body tenderservice.CancelTenderRequest true "Cancellation request"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/{tenderID}/cancel/ [post]
func (h *Handler) cancel(c *gin.Context) {
	tenderID, err := pulid.MustParse(c.Param("tenderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(tenderservice.CancelTenderRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFor(c)
	req.TenderID = tenderID

	if err = h.service.Cancel(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type recordResponseRequest struct {
	Action        tender.ResponseAction `json:"action"`
	DeclineReason string                `json:"declineReason"`
}

// @Summary Record a carrier's response received off-channel
// @ID recordTenderResponse
// @Tags Tenders
// @Accept json
// @Produce json
// @Param offerID path string true "Tender offer ID"
// @Param request body recordResponseRequest true "Manual response"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/offers/{offerID}/respond/ [post]
func (h *Handler) recordResponse(c *gin.Context) {
	offerID, err := pulid.MustParse(c.Param("offerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(recordResponseRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.RecordResponse(c.Request.Context(), &portservices.TenderResponseRequest{
		TenantInfo:    tenantInfoFor(c),
		OfferID:       offerID,
		Action:        req.Action,
		Source:        tender.ResponseSourceManual,
		DeclineReason: req.DeclineReason,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List a shipment's tenders
// @ID listTendersByShipment
// @Tags Tenders
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {array} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/by-shipment/{shipmentID}/ [get]
func (h *Handler) listByShipment(c *gin.Context) {
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entities, err := h.service.ListByShipment(
		c.Request.Context(),
		repositories.ListTendersByShipmentRequest{
			TenantInfo: tenantInfoFor(c),
			ShipmentID: shipmentID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

// @Summary Get the live tender for a move
// @ID getLiveTenderByMove
// @Tags Tenders
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Success 200 {object} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/by-move/{moveID}/live/ [get]
func (h *Handler) getLiveByMove(c *gin.Context) {
	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetLiveByMoveID(
		c.Request.Context(),
		repositories.GetLiveTenderByMoveRequest{
			TenantInfo: tenantInfoFor(c),
			MoveID:     moveID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
