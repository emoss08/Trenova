package customerstatementhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              customerstatementports.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service customerstatementports.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/accounts-receivable/customers/:customerID")
	api.GET(
		"/statement/pdf/",
		h.pm.RequirePermission(permission.ResourceAccountsReceivable.String(), permission.OpRead),
		h.pdf,
	)
}

func (h *Handler) pdf(c *gin.Context) {
	req := new(customerstatementports.StatementRequest)
	if err := c.ShouldBindQuery(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err := h.scope(c, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rendered, err := h.service.Render(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", fileutils.ContentDisposition("attachment", rendered.FileName))
	// The figures change with every payment; a cached copy would show a
	// balance that is no longer owed.
	c.Header("Cache-Control", "no-store")
	// Only ever a PDF: a sniffed response would hand organization-authored
	// template markup an app-origin document.
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/pdf", rendered.PDF)
}

// scope fills in who is asking and about which customer from the request
// itself, never from the body.
func (h *Handler) scope(c *gin.Context, req *customerstatementports.StatementRequest) error {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		return err
	}
	req.TenantInfo = tenantOf(c)
	req.CustomerID = customerID
	return nil
}

func tenantOf(c *gin.Context) pagination.TenantInfo {
	auth := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}
}
