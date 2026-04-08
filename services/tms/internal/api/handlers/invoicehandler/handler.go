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
	api.GET(
		"/:invoiceID/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/:invoiceID/post/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpUpdate),
		h.post,
	)
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
