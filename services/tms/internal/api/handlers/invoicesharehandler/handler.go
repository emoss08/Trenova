package invoicesharehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              invoiceports.InvoiceShareService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service invoiceports.InvoiceShareService
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
	api := rg.Group("/billing/invoices/:invoiceID/shares")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.share,
	)
	api.GET(
		"/candidates/",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.candidates,
	)
	api.GET(
		"/candidates/:userID",
		h.pm.RequirePermission(permission.ResourceInvoice.String(), permission.OpRead),
		h.candidate,
	)
}

type listSharesResponse struct {
	Shares []*invoice.InvoiceShare `json:"shares"`
}

type shareRequest struct {
	UserIDs []pulid.ID       `json:"userIds"`
	Note    string           `json:"note"`
	Tab     invoice.ShareTab `json:"tab"`
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	shares, err := h.service.List(c.Request.Context(), &invoiceports.ListInvoiceSharesRequest{
		TenantInfo: tenantInfo(authCtx),
		InvoiceID:  invoiceID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, listSharesResponse{Shares: shares})
}

func (h *Handler) share(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req shareRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Share(
		c.Request.Context(),
		&invoiceports.ShareInvoiceRequest{
			TenantInfo: tenantInfo(authCtx),
			InvoiceID:  invoiceID,
			UserIDs:    req.UserIDs,
			Note:       req.Note,
			Tab:        req.Tab,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) candidates(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewSelectQueryRequest(c, authCtx)
	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*invoiceports.ShareCandidate], error) {
			return h.service.ListCandidates(
				c.Request.Context(),
				&invoiceports.ListShareCandidatesRequest{
					TenantInfo: req.TenantInfo,
					InvoiceID:  invoiceID,
					Query:      req.Query,
					Pagination: req.Pagination,
				},
			)
		},
	)
}

func (h *Handler) candidate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	invoiceID, err := pulid.MustParse(c.Param("invoiceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	candidate, err := h.service.GetCandidate(
		c.Request.Context(),
		&invoiceports.GetShareCandidateRequest{
			TenantInfo: tenantInfo(authCtx),
			InvoiceID:  invoiceID,
			UserID:     userID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, candidate)
}

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
