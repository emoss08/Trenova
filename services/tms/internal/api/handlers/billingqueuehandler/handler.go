package billingqueuehandler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueuefilterpresetports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              billingqueueports.BillingQueueService
	Review               billingqueueports.BillingQueueReviewService
	Approval             billingqueueports.BillingQueueApprovalService
	Assist               billingqueueports.BillingQueueAssistService
	Packets              billingqueueports.PacketPreviewer
	PresetRepo           billingqueuefilterpresetports.BillingQueueFilterPresetRepository
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service    billingqueueports.BillingQueueService
	review     billingqueueports.BillingQueueReviewService
	approval   billingqueueports.BillingQueueApprovalService
	assist     billingqueueports.BillingQueueAssistService
	packets    billingqueueports.PacketPreviewer
	presetRepo billingqueuefilterpresetports.BillingQueueFilterPresetRepository
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
	formulas   formulavisibility.Gate
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		review:     p.Review,
		approval:   p.Approval,
		assist:     p.Assist,
		packets:    p.Packets,
		presetRepo: p.PresetRepo,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
		formulas:   formulavisibility.NewGate(p.PermissionMiddleware, p.Logger),
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
		"/recommendations/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.recommendations,
	)
	h.registerAssistRoutes(api)
	h.registerReviewRoutes(api)
	api.GET(
		"/:itemID/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.get,
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
	api.POST(
		"/:itemID/reassign-charge/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpUpdate,
		),
		h.reassignCharge,
	)
	api.GET(
		"/:itemID/packet-preview/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreview,
	)
	api.GET(
		"/:itemID/packet-preview/summary/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreviewSummary,
	)
	api.GET(
		"/:itemID/packet-preview/pages/:page/image/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreviewPage,
	)
	api.GET(
		"/:itemID/packet-preview/pages/:page/thumbnail/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreviewThumbnail,
	)
	api.GET(
		"/:itemID/packet-preview/checks/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreviewChecks,
	)
	api.PUT(
		"/:itemID/invoice-details/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpUpdate,
		),
		h.updateInvoiceDetails,
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
// @Param lane query []string false "Lanes to include" collectionFormat(multi)
// @Param minAmount query string false "Smallest amount, in the bill-to's currency"
// @Param maxAmount query string false "Largest amount, in the bill-to's currency"
// @Param createdBefore query int false "Only items queued before this Unix time"
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
			lanes, err := laneParams(c)
			if err != nil {
				return nil, err
			}
			minAmount, err := amountParam(c, "minAmount")
			if err != nil {
				return nil, err
			}
			maxAmount, err := amountParam(c, "maxAmount")
			if err != nil {
				return nil, err
			}
			var createdBefore *int64
			if before := helpers.QueryInt64(c, "createdBefore"); before > 0 {
				createdBefore = &before
			}
			result, err := h.service.List(
				c.Request.Context(),
				&billingqueueports.ListBillingQueueItemsRequest{
					Filter:        req,
					IncludePosted: helpers.QueryBool(c, "includePosted"),
					Lanes:         lanes,
					MinAmount:     minAmount,
					MaxAmount:     maxAmount,
					CreatedBefore: createdBefore,
				},
			)
			if err != nil {
				return nil, err
			}

			return h.visibleItemPage(c, result), nil
		},
	)
}

// @Summary Get billing queue stats
// @ID getBillingQueueStats
// @Tags Billing Queue
// @Produce json
// @Param dayStart query int false "Start of the caller's day, Unix seconds; adds the day's pulse"
// @Success 200 {object} billingqueueports.BillingQueueStats
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/stats/ [get]
func (h *Handler) stats(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	stats, err := h.service.GetStats(
		c.Request.Context(),
		&billingqueueports.GetBillingQueueStatsRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			DayStart:   helpers.QueryInt64(c, "dayStart"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, stats)
}

// @Summary List the batch actions the queue recommends
// @Description What a biller can clear in one action now: ready items, items that only need a biller, approved drafts to post, paperwork to ask drivers for, unbilled charges, possible duplicates, and the items Nova agreed with or questioned. Each names its items so a bulk action can act on exactly them.
// @ID listBillingQueueRecommendations
// @Tags Billing Queue
// @Produce json
// @Success 200 {array} billingqueueports.Recommendation
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/recommendations/ [get]
func (h *Handler) recommendations(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	canPost, err := h.pm.Allowed(c, permission.ResourceInvoice.String(), permission.OpPost)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	canAssign, err := h.pm.Allowed(c, permission.ResourceBillingQueue.String(), permission.OpAssign)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	recs, err := h.service.Recommendations(
		c.Request.Context(),
		&billingqueueports.ListRecommendationsRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			UserID:     authCtx.UserID,
			CanPost:    canPost,
			CanAssign:  canAssign,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, recs)
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
		&billingqueueports.GetBillingQueueItemByIDRequest{
			ItemID:                itemID,
			TenantInfo:            pagination.FromAuth(authCtx),
			ExpandShipmentDetails: helpers.QueryBool(c, "expandShipmentDetails"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, entity)
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
//
//nolint:govet // existing scoped variable reuse is local and behavior-preserving
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
		&billingqueueports.AssignBillerRequest{
			ItemID:     itemID,
			BillerID:   req.BillerID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, updated)
}

type updateChargesRequest struct {
	FormulaTemplateID *string                      `json:"formulaTemplateId"`
	TariffInputs      tarifftypes.Inputs           `json:"tariffInputs"`
	AdditionalCharges []*shipment.AdditionalCharge `json:"additionalCharges"`
	// ConvertAmountSplitsToPercent retries an edit that was refused because it
	// left an amount split that no longer adds up.
	ConvertAmountSplitsToPercent bool `json:"convertAmountSplitsToPercent"`
	ExpandShipmentDetails        bool `json:"expandShipmentDetails"`
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
//
//nolint:govet // existing scoped variable reuse is local and behavior-preserving
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
		&billingqueueports.UpdateChargesRequest{
			ItemID:            itemID,
			FormulaTemplateID: formulaTemplateID,
			TariffInputs:      req.TariffInputs,
			AdditionalCharges: req.AdditionalCharges,

			ConvertAmountSplitsToPercent: req.ConvertAmountSplitsToPercent,
			ExpandShipmentDetails:        req.ExpandShipmentDetails,
			TenantInfo:                   pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, updated)
}

type updateStatusRequest struct {
	Status              billingqueue.Status               `json:"status"              binding:"required"`
	ExceptionReasonCode *billingqueue.ExceptionReasonCode `json:"exceptionReasonCode"`
	ExceptionNotes      string                            `json:"exceptionNotes"`
	ReviewNotes         string                            `json:"reviewNotes"`
	CancelReason        string                            `json:"cancelReason"`
	HoldReasonCode      *billingqueue.HoldReasonCode      `json:"holdReasonCode"`
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
//
//nolint:govet // existing scoped variable reuse is local and behavior-preserving
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
		&billingqueueports.UpdateBillingQueueStatusRequest{
			ItemID:              itemID,
			NewStatus:           req.Status,
			ExceptionReasonCode: req.ExceptionReasonCode,
			ExceptionNotes:      req.ExceptionNotes,
			ReviewNotes:         req.ReviewNotes,
			CancelReason:        req.CancelReason,
			HoldReasonCode:      req.HoldReasonCode,
			TenantInfo:          pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.respondItem(c, http.StatusOK, updated)
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
		&billingqueuefilterpresetports.ListBillingQueueFilterPresetsRequest{
			UserID:     authCtx.UserID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{
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

	helpers.JSON(c, http.StatusCreated, created)
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
//
//nolint:govet // existing scoped variable reuse is local and behavior-preserving
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

	helpers.JSON(c, http.StatusOK, updated)
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
		&billingqueuefilterpresetports.DeleteBillingQueueFilterPresetRequest{
			PresetID:   presetID,
			UserID:     authCtx.UserID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type reassignChargeRequest struct {
	ChargeKind         shipmentvocab.ChargeAllocationKind `json:"chargeKind"`
	AdditionalChargeID pulid.ID                           `json:"additionalChargeId"`
	// Allocations divide the charge among payers. An empty list gives it back
	// whole to the shipment's payer.
	Allocations []*shipment.ChargeAllocation `json:"allocations"`
}

// @Summary Reassign who pays for one charge on a billing queue item's shipment
// @Description Moves a freight or accessorial charge, whole or split, to other payers while the shipment is in review. Payers who gain a share get a queue item and payers left with nothing have theirs canceled.
// @ID reassignBillingQueueCharge
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Param request body reassignChargeRequest true "Charge and its new allocations"
// @Success 200 {object} services.ReassignChargeResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/reassign-charge/ [post]
func (h *Handler) reassignCharge(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req reassignChargeRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.ReassignCharge(
		c.Request.Context(),
		&billingqueueports.ReassignChargeRequest{
			ItemID:             itemID,
			ChargeKind:         req.ChargeKind,
			AdditionalChargeID: req.AdditionalChargeID,
			Allocations:        req.Allocations,
			TenantInfo:         pagination.FromAuthAsUser(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	formulavisibility.Respond(
		c,
		h.formulas,
		http.StatusOK,
		result,
		(*billingqueueports.ReassignChargeResult).WithoutFormulaContent,
	)
}

func laneParams(c *gin.Context) ([]billingqueue.Lane, error) {
	raw := c.QueryArray("lane")
	lanes := make([]billingqueue.Lane, 0, len(raw))
	for _, value := range raw {
		lane := billingqueue.Lane(value)
		if !lane.IsValid() {
			return nil, errortypes.NewValidationError(
				"lane",
				errortypes.ErrInvalid,
				"Unknown billing queue lane {0}",
				value,
			)
		}
		lanes = append(lanes, lane)
	}

	return lanes, nil
}

type invoiceDetailsRequest struct {
	Version             int64                             `json:"version"`
	PONumber            string                            `json:"poNumber"`
	InvoiceMemo         string                            `json:"invoiceMemo"`
	PaymentTerm         string                            `json:"paymentTerm"`
	InvoiceDate         *int64                            `json:"invoiceDate"`
	PacketDocumentIDs   []pulid.ID                        `json:"packetDocumentIds"`
	PacketPicked        bool                              `json:"packetPicked"`
	PacketPageRotations []billingqueue.PacketPageRotation `json:"packetPageRotations"`
}

// @Summary Set what the invoice will say before it is made
// @Description The customer's PO, the note printed under the charges, a payment term and invoice date that override the customer's defaults, the shipment documents sent with the invoice as one packet, and the quarter turns given to their pages. Only before approval; after it, change the invoice itself.
// @ID updateBillingQueueInvoiceDetails
// @Tags Billing Queue
// @Accept json
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Success 200 {object} billingqueue.BillingQueueItem
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/invoice-details/ [put]
func (h *Handler) updateInvoiceDetails(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body invoiceDetailsRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	item, err := h.service.UpdateInvoiceDetails(
		c.Request.Context(),
		&billingqueueports.UpdateInvoiceDetailsRequest{
			ItemID:              itemID,
			TenantInfo:          pagination.FromAuth(authCtx),
			Version:             body.Version,
			PONumber:            body.PONumber,
			InvoiceMemo:         body.InvoiceMemo,
			PaymentTerm:         body.PaymentTerm,
			InvoiceDate:         body.InvoiceDate,
			PacketDocumentIDs:   body.PacketDocumentIDs,
			PacketPicked:        body.PacketPicked,
			PacketPageRotations: body.PacketPageRotations,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	formulavisibility.Respond(
		c,
		h.formulas,
		http.StatusOK,
		item,
		(*billingqueue.BillingQueueItem).WithoutFormulaContent,
	)
}

func amountParam(c *gin.Context, key string) (decimal.NullDecimal, error) {
	raw := c.Query(key)
	if raw == "" {
		return decimal.NullDecimal{}, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.IsNegative() {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			key,
			errortypes.ErrInvalid,
			"Enter an amount of zero or more",
		)
	}

	return decimal.NewNullDecimal(value), nil
}

// @Summary Preview an item's invoice packet
// @Description The one PDF approving the item would send: the invoice it would make, followed by the documents it carries. Nothing is saved.
// @ID previewBillingQueuePacket
// @Tags Billing Queue
// @Produce application/pdf
// @Param itemID path string true "Billing queue item ID"
// @Success 200 {file} binary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/ [get]
func (h *Handler) packetPreview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	preview, err := h.packets.PreviewPacket(
		c.Request.Context(),
		&billingqueueports.PacketPreviewRequest{
			ItemID:     itemID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="`+preview.FileName+`"`)
	c.Data(http.StatusOK, "application/pdf", preview.Content)
}

type packetPreviewSummary struct {
	FileName  string              `json:"fileName"`
	PageCount int                 `json:"pageCount"`
	Documents int                 `json:"documents"`
	Build     string              `json:"build"`
	Parts     []packetPreviewPart `json:"parts"`
}

type packetPreviewPart struct {
	Kind       billingqueueports.PacketPartKind `json:"kind"`
	DocumentID *string                          `json:"documentId"`
	Name       string                           `json:"name"`
	FirstPage  int                              `json:"firstPage"`
	LastPage   int                              `json:"lastPage"`
	Unsigned   bool                             `json:"unsigned"`
}

type packetPreviewChecks struct {
	Build      string `json:"build"`
	BlankPages []int  `json:"blankPages"`
}

func packetPartsOf(parts []billingqueueports.PacketPart) []packetPreviewPart {
	out := make([]packetPreviewPart, 0, len(parts))
	for _, part := range parts {
		var documentID *string
		if !part.DocumentID.IsNil() {
			id := part.DocumentID.String()
			documentID = &id
		}
		out = append(out, packetPreviewPart{
			Kind:       part.Kind,
			DocumentID: documentID,
			Name:       part.Name,
			FirstPage:  part.FirstPage,
			LastPage:   part.LastPage,
			Unsigned:   part.Unsigned,
		})
	}
	return out
}

// @Summary Assemble an item's invoice packet for viewing
// @Description Builds the packet approving the item would send and describes it, so its pages can be drawn one at a time. Nothing is saved.
// @ID summarizeBillingQueuePacket
// @Tags Billing Queue
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Success 200 {object} packetPreviewSummary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/summary/ [get]
func (h *Handler) packetPreviewSummary(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	preview, err := h.packets.PreviewPacket(
		c.Request.Context(),
		&billingqueueports.PacketPreviewRequest{
			ItemID:     itemID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, packetPreviewSummary{
		FileName:  preview.FileName,
		PageCount: preview.Pages,
		Documents: preview.Documents,
		Build:     preview.Build,
		Parts:     packetPartsOf(preview.Parts),
	})
}

// @Summary Draw a page of an item's invoice packet
// @Description One page of the packet approving the item would send, as an image. Nothing is saved.
// @ID drawBillingQueuePacketPage
// @Tags Billing Queue
// @Produce image/webp
// @Param itemID path string true "Billing queue item ID"
// @Param page path int true "Page number, from 1"
// @Success 200 {file} binary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/pages/{page}/image/ [get]
func (h *Handler) packetPreviewPage(c *gin.Context) {
	h.packetPageImage(c, h.packets.PreviewPacketPage)
}

// @Summary Draw a page of an item's invoice packet as a thumbnail
// @Description One page of the packet approving the item would send, as a small image for the page rail. Nothing is saved.
// @ID drawBillingQueuePacketThumbnail
// @Tags Billing Queue
// @Produce image/webp
// @Param itemID path string true "Billing queue item ID"
// @Param page path int true "Page number, from 1"
// @Success 200 {file} binary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/pages/{page}/thumbnail/ [get]
func (h *Handler) packetPreviewThumbnail(c *gin.Context) {
	h.packetPageImage(c, h.packets.PreviewPacketThumbnail)
}

func (h *Handler) packetPageImage(
	c *gin.Context,
	draw func(context.Context, *billingqueueports.PacketPageRequest) (*billingqueueports.PacketPage, error),
) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	page, err := strconv.Atoi(c.Param("page"))
	if err != nil || page < 1 {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"page",
				errortypes.ErrInvalid,
				"Page number must be a whole number from 1",
			),
		)
		return
	}

	image, err := draw(
		c.Request.Context(),
		&billingqueueports.PacketPageRequest{
			ItemID:     itemID,
			TenantInfo: pagination.FromAuth(authCtx),
			Page:       page,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, image.ContentType, image.Content)
}

// @Summary Check the pages of an item's invoice packet
// @Description Which document pages of the packet approving the item would send look blank. Nothing is saved.
// @ID checkBillingQueuePacket
// @Tags Billing Queue
// @Produce json
// @Param itemID path string true "Billing queue item ID"
// @Success 200 {object} packetPreviewChecks
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/checks/ [get]
func (h *Handler) packetPreviewChecks(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	checks, err := h.packets.PreviewPacketChecks(
		c.Request.Context(),
		&billingqueueports.PacketChecksRequest{
			ItemID:     itemID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, packetPreviewChecks{
		Build:      checks.Build,
		BlankPages: checks.BlankPages,
	})
}
