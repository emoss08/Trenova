package bankreceiptbatchhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              serviceports.BankReceiptBatchService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service serviceports.BankReceiptBatchService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/bank-receipt-batches")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:batchID/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpCreate),
		h.importBatch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET(
		"/sources/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.selectSources,
	)
}

func (h *Handler) list(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	items, err := h.service.List(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) get(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("batchID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.Get(
		c.Request.Context(),
		&serviceports.GetBankReceiptBatchRequest{
			BatchID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID:  auth.OrganizationID,
				BuID:   auth.BusinessUnitID,
				UserID: auth.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) selectSources(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, auth)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*repositories.BankReceiptBatchSourceOption], error) {
			return h.service.DistinctSources(c.Request.Context(), req)
		},
	)
}

func (h *Handler) importBatch(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := new(serviceports.ImportBankReceiptBatchRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}
	result, err := h.service.Import(c.Request.Context(), req, actorutil.FromAuthContext(auth))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}
