package shipmenthandler

import (
	"errors"
	"io"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              services.ShipmentService
	CommentService       services.ShipmentCommentService
	HoldService          services.ShipmentHoldService
	ImportAssistant      services.ShipmentImportAssistantService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service         services.ShipmentService
	commentService  services.ShipmentCommentService
	holdService     services.ShipmentHoldService
	importAssistant services.ShipmentImportAssistantService
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
	logger          *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service:         p.Service,
		commentService:  p.CommentService,
		holdService:     p.HoldService,
		importAssistant: p.ImportAssistant,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
		logger:          p.Logger.Named("api.shipment-handler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipments")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/ui-policy/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getUIPolicy,
	)
	api.GET(
		"/:shipmentID/billing-readiness/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getBillingReadiness,
	)
	api.GET(
		"/:shipmentID",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpCreate),
		h.create,
	)
	api.POST(
		"/calculate-totals/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.calculateTotals,
	)
	api.POST(
		"/duplicate/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpDuplicate),
		h.duplicate,
	)
	api.POST(
		"/check-for-duplicate-bols/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.checkForDuplicateBOLs,
	)
	api.POST(
		"/check-hazmat-segregation/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.checkHazmatSegregation,
	)
	api.POST(
		"/previous-rates/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getPreviousRates,
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
	api.GET(
		"/auto-cancel/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getAutoCancelableShipments,
	)
	api.POST(
		"/auto-cancel/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpCancel),
		h.autoCancelShipments,
	)
	api.GET(
		"/:shipmentID/comments/count/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.getCommentCount,
	)
	api.GET(
		"/:shipmentID/holds/",
		h.pm.RequirePermission(permission.ResourceShipmentHold.String(), permission.OpRead),
		h.listHolds,
	)
	api.GET(
		"/:shipmentID/holds/:holdID/",
		h.pm.RequirePermission(permission.ResourceShipmentHold.String(), permission.OpRead),
		h.getHold,
	)
	api.POST(
		"/:shipmentID/holds/",
		h.pm.RequirePermission(permission.ResourceShipmentHold.String(), permission.OpCreate),
		h.createHold,
	)
	api.PUT(
		"/:shipmentID/holds/:holdID/",
		h.pm.RequirePermission(permission.ResourceShipmentHold.String(), permission.OpUpdate),
		h.updateHold,
	)
	api.POST(
		"/:shipmentID/holds/:holdID/release/",
		h.pm.RequirePermission(permission.ResourceShipmentHold.String(), permission.OpUpdate),
		h.releaseHold,
	)
	api.GET(
		"/:shipmentID/comments/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.listComments,
	)
	api.POST(
		"/:shipmentID/comments/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpCreate),
		h.createComment,
	)
	api.PUT(
		"/:shipmentID/comments/:commentID/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.updateComment,
	)
	api.DELETE(
		"/:shipmentID/comments/:commentID/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpDelete),
		h.deleteComment,
	)
	api.POST(
		"/:shipmentID/cancel/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpCancel),
		h.cancel,
	)
	api.POST(
		"/:shipmentID/uncancel/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.uncancel,
	)
	api.POST(
		"/:shipmentID/transfer-ownership/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.transferOwnership,
	)
	api.PUT(
		"/:shipmentID/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpUpdate),
		h.update,
	)
}

// @Summary List shipments
// @Description Returns paginated shipments. Protected routes accept a Bearer token or an authenticated session cookie.
// @ID listShipments
// @Tags Shipments
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param expandShipmentDetails query bool false "Expand shipment details"
// @Param status query string false "Filter by shipment status"
// @Success 200 {object} pagination.Response[[]shipment.Shipment]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipment.Shipment], error) {
			return h.service.List(c.Request.Context(), &repositories.ListShipmentsRequest{
				Filter: req,
				ShipmentOptions: repositories.ShipmentOptions{
					ExpandShipmentDetails: helpers.QueryBool(c, "expandShipmentDetails"),
					Status:                helpers.QueryString(c, "status"),
				},
			})
		},
	)
}

// @Summary Get shipment UI policy
// @Description Returns shipment UI capability flags for the authenticated tenant.
// @ID getShipmentUIPolicy
// @Tags Shipments
// @Produce json
// @Success 200 {object} services.ShipmentUIPolicy
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/ui-policy/ [get]
func (h *Handler) getUIPolicy(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	policy, err := h.service.GetUIPolicy(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// @Summary Get shipment billing readiness
// @Description Returns billing-readiness details for a shipment, including required customer documents and resolved billing policy.
// @ID getShipmentBillingReadiness
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} services.ShipmentBillingReadiness
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/billing-readiness/ [get]
func (h *Handler) getBillingReadiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	readiness, err := h.service.GetBillingReadiness(c.Request.Context(), shipmentID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, readiness)
}

// @Summary Get a shipment
// @ID getShipment
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param expandShipmentDetails query bool false "Expand shipment details"
// @Param status query string false "Filter by shipment status"
// @Success 200 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetShipmentByIDRequest{
			ID: shipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: helpers.QueryBool(c, "expandShipmentDetails"),
				Status:                helpers.QueryString(c, "status"),
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a shipment
// @ID createShipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param request body shipment.Shipment true "Shipment payload"
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
	authctx.AddContextToRequest(authCtx, entity)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if entity.SourceDocumentID != "" {
		if _, err := pulid.Parse(entity.SourceDocumentID); err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError("sourceDocumentId", errortypes.ErrInvalid, "Invalid source document ID"))
			return
		}
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if entity.SourceDocumentID != "" && h.importAssistant != nil {
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

	c.JSON(http.StatusCreated, created)
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
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Calculate shipment totals
// @ID calculateShipmentTotals
// @Tags Shipments
// @Accept json
// @Produce json
// @Param request body shipment.Shipment true "Shipment payload"
// @Success 200 {object} repositories.ShipmentTotalsResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/calculate-totals/ [post]
func (h *Handler) calculateTotals(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(shipment.Shipment)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	authctx.AddContextToRequest(authCtx, entity)
	totals, err := h.service.CalculateTotals(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, totals)
}

// @Summary Duplicate a shipment
// @ID duplicateShipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param request body repositories.BulkDuplicateShipmentRequest true "Shipment duplication request"
// @Success 202 {object} repositories.ShipmentDuplicateWorkflowResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/duplicate/ [post]
func (h *Handler) duplicate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.BulkDuplicateShipmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	result, err := h.service.Duplicate(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) checkForDuplicateBOLs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.DuplicateBOLCheckRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	if err := h.service.CheckForDuplicateBOLs(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) checkHazmatSegregation(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.CheckHazmatSegregationRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	if err := h.service.CheckHazmatSegregation(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) getPreviousRates(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.GetPreviousRatesRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	entities, err := h.service.GetPreviousRates(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
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

	entities, err := h.service.GetDelayedShipments(
		c.Request.Context(),
		&repositories.GetDelayedShipmentsRequest{
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

	c.JSON(http.StatusOK, entities)
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

	entities, err := h.service.DelayShipments(
		c.Request.Context(),
		&repositories.DelayShipmentsRequest{
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

	c.JSON(http.StatusOK, entities)
}

// @Summary List auto-cancelable shipments
// @ID listAutoCancelableShipments
// @Tags Shipments
// @Produce json
// @Success 200 {array} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/auto-cancel/ [get]
func (h *Handler) getAutoCancelableShipments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entities, err := h.service.GetAutoCancelableShipments(
		c.Request.Context(),
		&repositories.GetAutoCancelableShipmentsRequest{
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

	c.JSON(http.StatusOK, entities)
}

// @Summary Auto-cancel eligible shipments
// @ID autoCancelShipments
// @Tags Shipments
// @Produce json
// @Success 200 {array} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/auto-cancel/ [post]
func (h *Handler) autoCancelShipments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entities, err := h.service.AutoCancelShipments(
		c.Request.Context(),
		&repositories.AutoCancelShipmentsRequest{
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

	c.JSON(http.StatusOK, entities)
}

// @Summary Cancel a shipment
// @ID cancelShipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body repositories.CancelShipmentRequest false "Shipment cancellation request"
// @Success 200 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/cancel/ [post]
func (h *Handler) cancel(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := &repositories.CancelShipmentRequest{
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}
	if err = c.ShouldBindJSON(req); err != nil && !errors.Is(err, io.EOF) {
		h.eh.HandleError(c, err)
		return
	}
	req.ShipmentID = shipmentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	actor := actorutil.FromAuthContext(authCtx)
	entity, err := h.service.Cancel(c.Request.Context(), req, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Uncancel a shipment
// @ID uncancelShipment
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/uncancel/ [post]
func (h *Handler) uncancel(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := &repositories.UncancelShipmentRequest{
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}

	actor := actorutil.FromAuthContext(authCtx)
	entity, err := h.service.Uncancel(c.Request.Context(), req, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Transfer shipment ownership
// @ID transferShipmentOwnership
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body repositories.TransferOwnershipRequest true "Ownership transfer request"
// @Success 200 {object} shipment.Shipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/transfer-ownership/ [post]
func (h *Handler) transferOwnership(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := &repositories.TransferOwnershipRequest{
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ShipmentID = shipmentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	actor := actorutil.FromAuthContext(authCtx)
	entity, err := h.service.TransferOwnership(c.Request.Context(), req, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
