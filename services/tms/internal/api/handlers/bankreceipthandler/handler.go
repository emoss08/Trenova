package bankreceipthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *bankreceiptservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *bankreceiptservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/bank-receipts")
	api.GET(
		"/summary/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.summary,
	)
	api.GET(
		"/exceptions/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.listExceptions,
	)
	api.GET(
		"/:receiptID/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.get,
	)
	api.GET(
		"/:receiptID/suggestions/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpRead),
		h.suggestions,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpCreate),
		h.importReceipt,
	)
	api.POST(
		"/:receiptID/match/",
		h.pm.RequirePermission(permission.ResourceBankReceipt.String(), permission.OpUpdate),
		h.match,
	)
}

func (h *Handler) summary(c *gin.Context) {
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
	summary, err := h.service.GetSummary(
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

func (h *Handler) listExceptions(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	items, err := h.service.ListExceptions(
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
	id, err := pulid.MustParse(c.Param("receiptID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Get(
		c.Request.Context(),
		&serviceports.GetBankReceiptRequest{
			ReceiptID: id,
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
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) suggestions(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("receiptID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	items, err := h.service.SuggestMatches(
		c.Request.Context(),
		&serviceports.GetBankReceiptRequest{
			ReceiptID: id,
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
	c.JSON(http.StatusOK, items)
}

func (h *Handler) importReceipt(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := new(serviceports.ImportBankReceiptRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}
	entity, err := h.service.Import(c.Request.Context(), req, actorutil.FromAuthContext(auth))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, entity)
}

func (h *Handler) match(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("receiptID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		PaymentID pulid.ID `json:"paymentId"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Match(
		c.Request.Context(),
		&serviceports.MatchBankReceiptRequest{
			ReceiptID: id,
			PaymentID: body.PaymentID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  auth.OrganizationID,
				BuID:   auth.BusinessUnitID,
				UserID: auth.UserID,
			},
		},
		actorutil.FromAuthContext(auth),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
