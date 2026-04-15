package customerpaymenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
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

	Service              serviceports.CustomerPaymentService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.CustomerPaymentService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/customer-payments")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceCustomerPayment.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:paymentID/",
		h.pm.RequirePermission(permission.ResourceCustomerPayment.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceCustomerPayment.String(), permission.OpCreate),
		h.postAndApply,
	)
	api.POST(
		"/:paymentID/apply/",
		h.pm.RequirePermission(permission.ResourceCustomerPayment.String(), permission.OpUpdate),
		h.applyUnapplied,
	)
	api.POST(
		"/:paymentID/reverse/",
		h.pm.RequirePermission(permission.ResourceCustomerPayment.String(), permission.OpUpdate),
		h.reverse,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	query := pagination.NewQueryOptions(c, authCtx)

	var customerID pulid.ID
	if raw := c.Query("customerId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		customerID = parsed
	}

	pagination.List(
		c,
		query,
		h.eh,
		func() (*pagination.ListResult[*customerpayment.Payment], error) {
			return h.service.List(c.Request.Context(), &repositories.ListCustomerPaymentsRequest{
				Filter:     query,
				CustomerID: customerID,
				Status:     customerpayment.Status(c.Query("status")),
			})
		},
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	paymentID, err := pulid.MustParse(c.Param("paymentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), &serviceports.GetCustomerPaymentRequest{
		PaymentID: paymentID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) postAndApply(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(serviceports.PostCustomerPaymentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	entity, err := h.service.PostAndApply(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *Handler) applyUnapplied(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	paymentID, err := pulid.MustParse(c.Param("paymentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(serviceports.ApplyCustomerPaymentRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.PaymentID = paymentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	entity, err := h.service.ApplyUnapplied(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) reverse(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	paymentID, err := pulid.MustParse(c.Param("paymentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(serviceports.ReverseCustomerPaymentRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.PaymentID = paymentID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	entity, err := h.service.Reverse(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
