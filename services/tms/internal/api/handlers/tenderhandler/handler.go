package tenderhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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
	api.POST(
		"/:tenderID/bounce/",
		h.pm.RequirePermission(permission.ResourceTender.String(), permission.OpUpdate),
		h.bounce,
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

	entity, err := h.service.Get(c.Request.Context(), tenderports.GetTenderByIDRequest{
		TenantInfo:    tenantInfoFor(c),
		TenderID:      tenderID,
		IncludeOffers: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Start a waterfall tender from a routing guide
// @ID createWaterfallTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param request body tendercreate.CreateWaterfallTenderRequest true "Waterfall tender request"
// @Success 201 {object} tendercreate.CreateWaterfallResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/waterfall/ [post]
func (h *Handler) createWaterfall(c *gin.Context) {
	req := new(tendercreate.CreateWaterfallTenderRequest)
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

	helpers.JSON(c, http.StatusCreated, entity)
}

// @Summary Start a spot tender with an ad-hoc carrier list
// @ID createSpotTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param request body tendercreate.CreateSpotTenderRequest true "Spot tender request"
// @Success 201 {object} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/spot/ [post]
func (h *Handler) createSpot(c *gin.Context) {
	req := new(tendercreate.CreateSpotTenderRequest)
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

	helpers.JSON(c, http.StatusCreated, entity)
}

// @Summary Cancel a live tender
// @ID cancelTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param tenderID path string true "Tender ID"
// @Param request body tendercancel.CancelTenderRequest true "Cancellation request"
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

	req := new(tendercancel.CancelTenderRequest)
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
	CounterRate   decimal.Decimal       `json:"counterRate"`
	CounterNote   string                `json:"counterNote"`
}

// @Summary Record a carrier's response received off-channel
// @Description Accept, Decline or Counter an offer. On an offer holding a counter, Accept takes the counter and Decline refuses it; taking one below the margin floor needs the approve permission on tenders.
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

	if err = h.service.RecordResponse(c.Request.Context(), &tenderports.TenderResponseRequest{
		Actor:         requestActorFromAuthContext(authctx.GetAuthContext(c)),
		TenantInfo:    tenantInfoFor(c),
		OfferID:       offerID,
		Action:        req.Action,
		Source:        tender.ResponseSourceManual,
		DeclineReason: req.DeclineReason,
		CounterRate:   req.CounterRate,
		CounterNote:   req.CounterNote,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type bounceRequest struct {
	Reason string `json:"reason"`
}

// @Summary Record a carrier backing out of a tender it accepted
// @Description Takes the carrier off the move, voids its rate confirmation and sends the tender on to the carriers the acceptance cut short, in rank order.
// @ID bounceTender
// @Tags Tenders
// @Accept json
// @Produce json
// @Param tenderID path string true "Tender ID"
// @Param request body bounceRequest true "Why the carrier backed out"
// @Success 200 {object} tender.Tender
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /tenders/{tenderID}/bounce/ [post]
func (h *Handler) bounce(c *gin.Context) {
	tenderID, err := pulid.MustParse(c.Param("tenderID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(bounceRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Bounce(c.Request.Context(), &tenderbounce.BounceTenderRequest{
		TenantInfo: tenantInfoFor(c),
		TenderID:   tenderID,
		Reason:     req.Reason,
		Source:     tender.ResponseSourceManual,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

func requestActorFromAuthContext(authCtx *authctx.AuthContext) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
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
		tenderports.ListTendersByShipmentRequest{
			TenantInfo: tenantInfoFor(c),
			ShipmentID: shipmentID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entities)
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
		tenderports.GetLiveTenderByMoveRequest{
			TenantInfo: tenantInfoFor(c),
			MoveID:     moveID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}
