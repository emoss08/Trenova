package billingqueuehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.BillingQueueService
	PresetRepo           repositories.BillingQueueFilterPresetRepository
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service    services.BillingQueueService
	presetRepo repositories.BillingQueueFilterPresetRepository
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		presetRepo: p.PresetRepo,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing-queue")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.GET(
		"/stats/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.stats,
	)
	api.GET(
		"/:itemID/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.POST(
		"/transfer/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpCreate,
		),
		h.transfer,
	)
	api.PUT(
		"/:itemID/assign/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpAssign,
		),
		h.assign,
	)
	api.PUT(
		"/:itemID/status/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpUpdate,
		),
		h.updateStatus,
	)
	api.PUT(
		"/:itemID/charges/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpUpdate,
		),
		h.updateCharges,
	)

	presets := api.Group("/filter-presets")
	presets.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.listFilterPresets,
	)
	presets.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.createFilterPreset,
	)
	presets.PUT(
		"/:presetId/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.updateFilterPreset,
	)
	presets.DELETE(
		"/:presetId/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.deleteFilterPreset,
	)
}

// @Summary List billing queue items
// @ID listBillingQueueItems
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]billingqueue.BillingQueueItem]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListBillingQueueItemsRequest{
					Filter:        req,
					IncludePosted: helpers.QueryBool(c, "includePosted"),
				},
			)
		},
	)
}

// @Summary Get billing queue stats
// @ID getBillingQueueStats
// @Tags Billing Queue
// @Produce json
// @Success 200 {object} services.BillingQueueStats
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/stats/ [get]
func (h *Handler) stats(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	stats, err := h.service.GetStats(
		c.Request.Context(),
		&repositories.GetBillingQueueStatsRequest{
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

	c.JSON(http.StatusOK, stats)
}

// @Summary Get a billing queue item
// @ID getBillingQueueItem
// @Tags Billing Queue
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param expandShipmentDetails query bool false "Expand shipment details"
// @Success 200 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetBillingQueueItemByIDRequest{
			ItemID: itemID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ExpandShipmentDetails: helpers.QueryBool(c, "expandShipmentDetails"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type transferRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId" binding:"required"`
	BillType   billingqueue.BillType `json:"billType"`
}

// @Summary Transfer a shipment to billing queue
// @ID transferToBillingQueue
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param request body transferRequest true "Transfer request payload"
// @Success 201 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/transfer/ [post]
func (h *Handler) transfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.BillType == "" {
		req.BillType = billingqueue.BillTypeInvoice
	}

	created, err := h.service.TransferToBilling(
		c.Request.Context(),
		&services.TransferToBillingRequest{
			ShipmentID: req.ShipmentID,
			BillType:   req.BillType,
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

	c.JSON(http.StatusCreated, created)
}

type assignRequest struct {
	BillerID pulid.ID `json:"billerId" binding:"required"`
}

// @Summary Assign a biller to a billing queue item
// @ID assignBillingQueueBiller
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param request body assignRequest true "Assign biller payload"
// @Success 200 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/assign/ [put]
func (h *Handler) assign(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req assignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.AssignBiller(
		c.Request.Context(),
		&services.AssignBillerRequest{
			ItemID:   itemID,
			BillerID: req.BillerID,
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

	c.JSON(http.StatusOK, updated)
}

type updateChargesRequest struct {
	FormulaTemplateID *string                      `json:"formulaTemplateId"`
	BaseRate          *string                      `json:"baseRate"`
	AdditionalCharges []*shipment.AdditionalCharge `json:"additionalCharges"`
}

// @Summary Update charges on a billing queue item
// @ID updateBillingQueueCharges
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param request body updateChargesRequest true "Update charges payload"
// @Success 200 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/charges/ [put]
func (h *Handler) updateCharges(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req updateChargesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var baseRate *decimal.Decimal
	if req.BaseRate != nil {
		d, dErr := decimal.NewFromString(*req.BaseRate)
		if dErr != nil {
			h.eh.HandleError(c, dErr)
			return
		}
		baseRate = &d
	}

	var formulaTemplateID *pulid.ID
	if req.FormulaTemplateID != nil {
		id, idErr := pulid.MustParse(*req.FormulaTemplateID)
		if idErr != nil {
			h.eh.HandleError(c, idErr)
			return
		}
		formulaTemplateID = &id
	}

	updated, err := h.service.UpdateCharges(
		c.Request.Context(),
		&services.UpdateChargesRequest{
			ItemID:            itemID,
			FormulaTemplateID: formulaTemplateID,
			BaseRate:          baseRate,
			AdditionalCharges: req.AdditionalCharges,
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

	c.JSON(http.StatusOK, updated)
}

type updateStatusRequest struct {
	Status              billingqueue.Status               `json:"status"              binding:"required"`
	ExceptionReasonCode *billingqueue.ExceptionReasonCode `json:"exceptionReasonCode"`
	ExceptionNotes      string                            `json:"exceptionNotes"`
	ReviewNotes         string                            `json:"reviewNotes"`
	CancelReason        string                            `json:"cancelReason"`
}

// @Summary Update billing queue item status
// @ID updateBillingQueueStatus
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param request body updateStatusRequest true "Update status payload"
// @Success 200 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/status/ [put]
func (h *Handler) updateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateStatus(
		c.Request.Context(),
		&services.UpdateBillingQueueStatusRequest{
			ItemID:              itemID,
			NewStatus:           req.Status,
			ExceptionReasonCode: req.ExceptionReasonCode,
			ExceptionNotes:      req.ExceptionNotes,
			ReviewNotes:         req.ReviewNotes,
			CancelReason:        req.CancelReason,
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

	c.JSON(http.StatusOK, updated)
}

// @Summary List billing queue filter presets
// @ID listBillingQueueFilterPresets
// @Tags Billing Queue
// @Produce json
// @Success 200 {array} billingqueuefilterpreset.BillingQueueFilterPreset
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/filter-presets/ [get]
func (h *Handler) listFilterPresets(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	presets, err := h.presetRepo.ListByUserID(
		c.Request.Context(),
		&repositories.ListBillingQueueFilterPresetsRequest{
			UserID: authCtx.UserID,
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

	c.JSON(http.StatusOK, gin.H{
		"results": presets,
		"count":   len(presets),
	})
}

type filterPresetRequest struct {
	Name      string         `json:"name"      binding:"required"`
	Filters   map[string]any `json:"filters"   binding:"required"`
	IsDefault bool           `json:"isDefault"`
}

// @Summary Create a billing queue filter preset
// @ID createBillingQueueFilterPreset
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param request body filterPresetRequest true "Filter preset payload"
// @Success 201 {object} billingqueuefilterpreset.BillingQueueFilterPreset
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/filter-presets/ [post]
func (h *Handler) createFilterPreset(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req filterPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := &billingqueuefilterpreset.BillingQueueFilterPreset{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Name:           req.Name,
		Filters:        req.Filters,
		IsDefault:      req.IsDefault,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		h.eh.HandleError(c, multiErr)
		return
	}

	created, err := h.presetRepo.Create(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a billing queue filter preset
// @ID updateBillingQueueFilterPreset
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param presetId path string true "Filter preset ID"
// @Param request body filterPresetRequest true "Filter preset payload"
// @Success 200 {object} billingqueuefilterpreset.BillingQueueFilterPreset
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/filter-presets/{presetId}/ [put]
func (h *Handler) updateFilterPreset(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	presetID, err := pulid.MustParse(c.Param("presetId"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req filterPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := &billingqueuefilterpreset.BillingQueueFilterPreset{
		ID:             presetID,
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Name:           req.Name,
		Filters:        req.Filters,
		IsDefault:      req.IsDefault,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		h.eh.HandleError(c, multiErr)
		return
	}

	updated, err := h.presetRepo.Update(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a billing queue filter preset
// @ID deleteBillingQueueFilterPreset
// @Tags Billing Queue
// @Param presetId path string true "Filter preset ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/filter-presets/{presetId}/ [delete]
func (h *Handler) deleteFilterPreset(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	presetID, err := pulid.MustParse(c.Param("presetId"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.presetRepo.Delete(
		c.Request.Context(),
		&repositories.DeleteBillingQueueFilterPresetRequest{
			PresetID: presetID,
			UserID:   authCtx.UserID,
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

	c.Status(http.StatusNoContent)
}
