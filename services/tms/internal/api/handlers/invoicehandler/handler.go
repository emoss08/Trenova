package invoicehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.InvoiceService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.InvoiceService
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
	api := rg.Group("/billing/invoices")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/from-shipments/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpCreate),
		h.createFromShipments,
	)
	api.GET(
		"/:invoiceID/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.get,
	)
	api.PATCH(
		"/:invoiceID/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate),
		h.updateDraft,
	)
	api.POST(
		"/:invoiceID/preview/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate),
		h.preview,
	)
	api.POST(
		"/:invoiceID/generate-pdf/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate),
		h.generatePDF,
	)
	api.GET(
		"/:invoiceID/send-plan/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.sendPlan,
	)
	api.POST(
		"/:invoiceID/send/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpSubmit),
		h.send,
	)
	api.POST(
		"/:invoiceID/resend/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpSubmit),
		h.send,
	)
	api.GET(
		"/:invoiceID/email-attempts/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.emailAttempts,
	)
	api.POST(
		"/:invoiceID/post/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate),
		h.post,
	)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing/invoices")
	api.GET(
		"/shared-documents/:token/download/",
		h.downloadSharedDocument,
	)
}

type createFromShipmentsRequest struct {
	ShipmentIDs []pulid.ID `json:"shipmentIds"`
}

type updateDraftRequest struct {
	Memo                   *string     `json:"memo"`
	RemittanceInstructions *string     `json:"remittanceInstructions"`
	EmailSubject           *string     `json:"emailSubject"`
	EmailBody              *string     `json:"emailBody"`
	EmailTo                *[]string   `json:"emailTo"`
	EmailCC                *[]string   `json:"emailCc"`
	EmailBCC               *[]string   `json:"emailBcc"`
	AttachmentIDs          *[]pulid.ID `json:"attachmentIds"`
}

// @Summary List invoices
// @ID listInvoices
// @Tags Invoices
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]invoice.Invoice]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoices/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*invoice.Invoice], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListInvoicesRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get an invoice
// @ID getInvoice
// @Tags Invoices
// @Produce json
// @Param invoiceID path string true "Invoice ID"
// @Success 200 {object} invoice.Invoice
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoices/{invoiceID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetInvoiceByIDRequest{
		ID: invoiceID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createFromShipments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	var req createFromShipmentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.CreateFromShipments(
		c.Request.Context(),
		&services.CreateInvoiceFromShipmentsRequest{
			ShipmentIDs: req.ShipmentIDs,
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

	c.JSON(http.StatusCreated, entity)
}

func (h *Handler) updateDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var req updateDraftRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateDraft(
		c.Request.Context(),
		&services.UpdateInvoiceDraftRequest{
			InvoiceID:              invoiceID,
			TenantInfo:             tenantInfo(authCtx),
			Memo:                   req.Memo,
			RemittanceInstructions: req.RemittanceInstructions,
			EmailSubject:           req.EmailSubject,
			EmailBody:              req.EmailBody,
			EmailTo:                req.EmailTo,
			EmailCC:                req.EmailCC,
			EmailBCC:               req.EmailBCC,
			AttachmentIDs:          req.AttachmentIDs,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.RenderPreview(
		c.Request.Context(),
		&services.InvoicePreviewRequest{InvoiceID: invoiceID, TenantInfo: tenantInfo(authCtx)},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", `inline; filename="`+result.FileName+`"`)
	c.Data(http.StatusOK, result.ContentType, result.Content)
}

func (h *Handler) generatePDF(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.GeneratePDF(
		c.Request.Context(),
		&services.InvoicePreviewRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo(authCtx),
			BaseURL:    baseURL(c),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) sendPlan(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	plan, err := h.service.PlanSend(
		c.Request.Context(),
		&services.InvoiceSendPlanRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo(authCtx),
			BaseURL:    baseURL(c),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) send(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.Send(
		c.Request.Context(),
		&services.InvoiceSendRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo(authCtx),
			BaseURL:    baseURL(c),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) emailAttempts(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*invoice.EmailAttempt], error) {
			return h.service.ListEmailAttempts(
				c.Request.Context(),
				repositories.ListInvoiceEmailAttemptsRequest{
					InvoiceID:  invoiceID,
					TenantInfo: req.TenantInfo,
					Filter:     req,
				},
			)
		},
	)
}

func (h *Handler) downloadSharedDocument(c *gin.Context) {
	result, err := h.service.DownloadSharedDocument(
		c.Request.Context(),
		&services.DownloadInvoiceDocumentRequest{Token: c.Param("token")},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", result.ContentDisposition)
	c.Data(http.StatusOK, result.ContentType, result.Body)
}

// @Summary Post an invoice
// @ID postInvoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param invoiceID path string true "Invoice ID"
// @Success 200 {object} invoice.Invoice
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ValidationError
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/invoices/{invoiceID}/post/ [post]
func (h *Handler) post(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Post(
		c.Request.Context(),
		&services.PostInvoiceRequest{
			InvoiceID: invoiceID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			TriggeredBy: "manual",
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func baseURL(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}
