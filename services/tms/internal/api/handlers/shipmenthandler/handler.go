//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package shipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Reader               shipmentserviceports.ShipmentReader
	Writer               shipmentserviceports.ShipmentWriter
	Lifecycle            shipmentserviceports.ShipmentLifecycleService
	Readiness            shipmentserviceports.ShipmentBillingReadinessReader
	Transfer             shipmentserviceports.ShipmentBillingTransferService
	AutoRater            shipmentserviceports.ShipmentAutoRater
	Loading              shipmentserviceports.ShipmentLoadingPlanner
	CommentService       shipmentcommentports.ShipmentCommentService
	HoldService          shipmentholdports.ShipmentHoldService
	ImportAssistant      shipmentimportassistantports.ShipmentImportAssistantService
	PermitService        permitports.PermitService
	Etas                 shipmenttrackingports.ShipmentEtaReader
	CustomerTracking     customertrackingports.CustomerTrackingService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	reader          shipmentserviceports.ShipmentReader
	writer          shipmentserviceports.ShipmentWriter
	lifecycle       shipmentserviceports.ShipmentLifecycleService
	readiness       shipmentserviceports.ShipmentBillingReadinessReader
	transfer        shipmentserviceports.ShipmentBillingTransferService
	autorater       shipmentserviceports.ShipmentAutoRater
	loading         shipmentserviceports.ShipmentLoadingPlanner
	commentService  shipmentcommentports.ShipmentCommentService
	holdService     shipmentholdports.ShipmentHoldService
	importAssistant shipmentimportassistantports.ShipmentImportAssistantService
	permitService   permitports.PermitService
	etas            shipmenttrackingports.ShipmentEtaReader
	tracking        customertrackingports.CustomerTrackingService
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
	formulas        formulavisibility.Gate
	logger          *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		reader:          p.Reader,
		writer:          p.Writer,
		lifecycle:       p.Lifecycle,
		readiness:       p.Readiness,
		transfer:        p.Transfer,
		autorater:       p.AutoRater,
		loading:         p.Loading,
		commentService:  p.CommentService,
		holdService:     p.HoldService,
		importAssistant: p.ImportAssistant,
		permitService:   p.PermitService,
		etas:            p.Etas,
		tracking:        p.CustomerTracking,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
		formulas:        formulavisibility.NewGate(p.PermissionMiddleware, p.Logger),
		logger:          p.Logger.Named("api.shipment-handler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) { //nolint:funlen // legacy workflow
	api := rg.Group("/shipments")
	api.GET(
		"/:shipmentID/eta/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getEta,
	)
	api.GET(
		"/:shipmentID/customer-tracking/preview/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.previewCustomerTracking,
	)
	api.GET(
		"/:shipmentID/permit-assessment/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getPermitAssessment,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpCreate),
		h.create,
	)
	api.GET(
		"/delayed/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getDelayedShipments,
	)
	api.POST(
		"/delay/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.delayShipments,
	)
	api.PUT(
		"/:shipmentID/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.update,
	)
}

// @Summary Get shipment ETA
// @Description Returns the stored projected arrival at every open stop of a shipment and the shipment's verdict, taken from its worst open stop.
// @ID getShipmentEta
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} shipmenttrackingports.ShipmentEta
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/eta/ [get]
func (h *Handler) getEta(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	etas, err := h.etas.EtasByShipmentIDs(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		[]pulid.ID{shipmentID},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	eta := etas[shipmentID]
	if eta == nil {
		eta = shipmenttrackingports.UnknownShipmentEta(shipmentID)
	}
	helpers.JSON(c, http.StatusOK, eta)
}

// @Summary Preview a shipment's customer tracking page
// @Description Returns what the shipment's customer would see through a tracking link opened now, built from the customer's tracking settings. No link is created and no view is recorded.
// @ID previewShipmentCustomerTracking
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} customertrackingports.PublicTrackingView
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/customer-tracking/preview/ [get]
func (h *Handler) previewCustomerTracking(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	view, err := h.tracking.PreviewForShipment(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		shipmentID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, view)
}

// @Summary Get shipment permit assessment
// @Description Returns the oversize/overweight assessment for a shipment: the measured envelope, every jurisdiction on the route with its limits and headroom, derived permit requirements, escorts, restrictions, estimated fees and the earliest feasible pickup.
// @ID getShipmentPermitAssessment
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} services.PermitAssessment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/permit-assessment/ [get]
func (h *Handler) getPermitAssessment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.reader.Get(
		c.Request.Context(),
		&shipmentports.GetShipmentByIDRequest{
			ID:         shipmentID,
			TenantInfo: pagination.FromAuth(authCtx),
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	assessment, err := h.permitService.Assess(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, assessment)
}

// @Summary Create a shipment
// @ID createShipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param request body shipment.Shipment true "Shipment payload"
// @Param Idempotency-Key header string false "Client key that makes a retried create return the first result instead of booking another shipment"
// @Success 201 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(shipment.Shipment)
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if entity.SourceDocumentID != "" {
		if _, err := pulid.Parse(entity.SourceDocumentID); err != nil {
			h.eh.HandleError(
				c,
				errortypes.NewValidationError(
					"sourceDocumentId",
					errortypes.ErrInvalid,
					"Invalid source document ID",
				),
			)
			return
		}
	}

	entity.IdempotencyKey, _ = idempotency.KeyFrom(c.Request.Context())

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.writer.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if entity.SourceDocumentID != "" {
		if completeErr := h.importAssistant.CompleteHistory(
			c.Request.Context(),
			entity.SourceDocumentID,
			pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		); completeErr != nil {
			h.logger.Warn(
				"failed to complete shipment import assistant history after shipment creation",
				zap.String("sourceDocumentId", entity.SourceDocumentID),
				zap.Stringer("shipmentId", created.ID),
				zap.Error(completeErr),
			)
		}
	}

	h.respondShipment(c, http.StatusCreated, created)
}

// @Summary Update a shipment
// @ID updateShipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body shipment.Shipment true "Shipment payload"
// @Success 200 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(shipment.Shipment)
	entity.ID = shipmentID
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.writer.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondShipment(c, http.StatusOK, updated)
}

// @Summary List delayed shipments
// @ID listDelayedShipments
// @Tags Shipments
// @Produce json
// @Success 200 {array} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/delayed/ [get]
func (h *Handler) getDelayedShipments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entities, err := h.lifecycle.GetDelayedShipments(
		c.Request.Context(),
		&shipmentports.GetDelayedShipmentsRequest{
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondShipments(c, entities)
}

// @Summary Delay eligible shipments
// @ID delayShipments
// @Tags Shipments
// @Produce json
// @Success 200 {array} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/delay/ [post]
func (h *Handler) delayShipments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entities, err := h.lifecycle.DelayShipments(
		c.Request.Context(),
		&shipmentports.DelayShipmentsRequest{
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondShipments(c, entities)
}
