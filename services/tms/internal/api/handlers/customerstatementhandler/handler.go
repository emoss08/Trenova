package customerstatementhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
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
	read := h.pm.RequirePermission(
		permission.ResourceAccountsReceivable.String(),
		permission.OpRead,
	)
	work := h.pm.RequirePermission(
		permission.ResourceAccountsReceivable.String(),
		permission.OpUpdate,
	)

	api.GET("/statement/pdf/", read, h.pdf)
	api.POST("/statement/send/", work, h.send)
	api.GET("/statements/", read, h.history)
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

func (h *Handler) send(c *gin.Context) {
	req := new(customerstatementports.SendRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err := h.scope(c, &req.StatementRequest); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Send(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) history(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	sent, err := h.service.ListSent(c.Request.Context(), customerstatementports.ListSentRequest{
		TenantInfo: tenantOf(c),
		CustomerID: customerID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"statements": sent})
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
