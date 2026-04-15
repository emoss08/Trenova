package accountsreceivablehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *accountsreceivableservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *accountsreceivableservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/accounts-receivable")
	api.GET(
		"/aging/",
		h.pm.RequirePermission(permission.ResourceAccountsReceivable.String(), permission.OpRead),
		h.aging,
	)
	api.GET(
		"/open-items/",
		h.pm.RequirePermission(permission.ResourceAccountsReceivable.String(), permission.OpRead),
		h.openItems,
	)
	api.GET(
		"/customers/:customerID/ledger/",
		h.pm.RequirePermission(permission.ResourceAccountsReceivable.String(), permission.OpRead),
		h.ledger,
	)
	api.GET(
		"/customers/:customerID/statement/",
		h.pm.RequirePermission(permission.ResourceAccountsReceivable.String(), permission.OpRead),
		h.statement,
	)
}

func (h *Handler) aging(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	asOfDate := int64(0)
	if raw := c.Query("asOfDate"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		asOfDate = parsed
	}
	summary, err := h.service.GetAgingSummary(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		asOfDate,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) ledger(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entries, err := h.service.ListCustomerLedger(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		customerID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) openItems(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	asOfDate := int64(0)
	if raw := c.Query("asOfDate"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		asOfDate = parsed
	}

	var customerID pulid.ID
	if raw := c.Query("customerId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		customerID = parsed
	}

	items, err := h.service.ListOpenItems(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		customerID,
		asOfDate,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) statement(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	asOfDate := int64(0)
	if raw := c.Query("asOfDate"); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			h.eh.HandleError(c, parseErr)
			return
		}
		asOfDate = parsed
	}

	startDate := int64(0)
	if raw := c.Query("startDate"); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			h.eh.HandleError(c, parseErr)
			return
		}
		startDate = parsed
	}

	statement, err := h.service.GetCustomerStatement(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		},
		customerID,
		startDate,
		asOfDate,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, statement)
}
