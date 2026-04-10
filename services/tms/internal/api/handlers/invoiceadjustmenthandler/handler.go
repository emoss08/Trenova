package invoiceadjustmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.InvoiceAdjustmentService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.InvoiceAdjustmentService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing/invoice-adjustments")
	api.POST("/drafts/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate), h.createDraft)
	api.PATCH("/drafts/:adjustmentID/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate), h.updateDraft)
	api.POST("/drafts/:adjustmentID/preview/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.previewDraft)
	api.POST("/drafts/:adjustmentID/submit/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate), h.submitDraft)
	api.POST("/preview/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.preview)
	api.POST("/submit/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate), h.submit)
	api.POST("/bulk-preview/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.bulkPreview)
	api.POST("/bulk-submit/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate), h.bulkSubmit)
	api.GET("/summary/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.summary)
	api.GET("/approvals/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.approvals)
	api.GET("/reconciliation-exceptions/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.reconciliationExceptions)
	api.GET("/batches/:batchID/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.getBatch)
	api.GET("/batches/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.listBatches)
	api.GET("/correction-groups/:correctionGroupID/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.groupLineage)
	api.GET("/:adjustmentID/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.get)
	api.POST("/:adjustmentID/approve/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpApprove), h.approve)
	api.POST("/:adjustmentID/reject/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpApprove), h.reject)
	api.GET("/:adjustmentID/lineage/", h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead), h.lineage)
}

func (h *Handler) createDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	var body struct {
		InvoiceID pulid.ID `json:"invoiceId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.CreateDraft(c.Request.Context(), &services.CreateDraftInvoiceAdjustmentRequest{
		InvoiceID: body.InvoiceID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(services.UpdateDraftInvoiceAdjustmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.AdjustmentID = adjustmentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	resp, err := h.service.UpdateDraft(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) previewDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.PreviewDraft(c.Request.Context(), &services.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: adjustmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) submitDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.SubmitDraft(c.Request.Context(), &services.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: adjustmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(services.InvoiceAdjustmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	resp, err := h.service.Preview(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) submit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(services.InvoiceAdjustmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	resp, err := h.service.Submit(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) bulkPreview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(services.InvoiceAdjustmentBulkRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	resp, err := h.service.BulkPreview(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) bulkSubmit(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(services.InvoiceAdjustmentBulkRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	resp, err := h.service.BulkSubmit(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) summary(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	resp, err := h.service.GetOperationsSummary(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) approvals(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*invoiceadjustment.ApprovalQueueItem], error) {
		return h.service.ListApprovals(c.Request.Context(), *req)
	})
}

func (h *Handler) reconciliationExceptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*invoiceadjustment.ReconciliationQueueItem], error) {
		return h.service.ListReconciliationExceptions(c.Request.Context(), *req)
	})
}

func (h *Handler) getBatch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	batchID, err := pulid.MustParse(c.Param("batchID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.GetBatch(c.Request.Context(), batchID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listBatches(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*invoiceadjustment.BatchQueueItem], error) {
		return h.service.ListBatches(c.Request.Context(), *req)
	})
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.GetDetail(c.Request.Context(), &services.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: adjustmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) approve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.Approve(c.Request.Context(), &services.ApproveInvoiceAdjustmentRequest{
		AdjustmentID: adjustmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) reject(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.Reject(c.Request.Context(), &services.RejectInvoiceAdjustmentRequest{
		AdjustmentID: adjustmentID,
		Reason:       body.Reason,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) lineage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	adjustmentID, err := pulid.MustParse(c.Param("adjustmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	detail, err := h.service.GetDetail(c.Request.Context(), &services.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: adjustmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.GetLineage(c.Request.Context(), &services.GetInvoiceAdjustmentLineageRequest{
		CorrectionGroupID: detail.CorrectionGroupID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) groupLineage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	groupID, err := pulid.MustParse(c.Param("correctionGroupID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.GetLineage(c.Request.Context(), &services.GetInvoiceAdjustmentLineageRequest{
		CorrectionGroupID: groupID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
