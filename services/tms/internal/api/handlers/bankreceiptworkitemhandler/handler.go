package bankreceiptworkitemhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *bankreceiptworkitemservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *bankreceiptworkitemservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/bank-receipt-work-items")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceBankReceiptWorkItem.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:workItemID/",
		h.pm.RequirePermission(permission.ResourceBankReceiptWorkItem.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/:workItemID/assign/",
		h.pm.RequirePermission(
			permission.ResourceBankReceiptWorkItem.String(),
			permission.OpUpdate,
		),
		h.assign,
	)
	api.POST(
		"/:workItemID/start-review/",
		h.pm.RequirePermission(
			permission.ResourceBankReceiptWorkItem.String(),
			permission.OpUpdate,
		),
		h.startReview,
	)
	api.POST(
		"/:workItemID/resolve/",
		h.pm.RequirePermission(
			permission.ResourceBankReceiptWorkItem.String(),
			permission.OpUpdate,
		),
		h.resolve,
	)
	api.POST(
		"/:workItemID/dismiss/",
		h.pm.RequirePermission(
			permission.ResourceBankReceiptWorkItem.String(),
			permission.OpUpdate,
		),
		h.dismiss,
	)
}

func (h *Handler) list(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	items, err := h.service.ListActive(
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
	id, err := pulid.MustParse(c.Param("workItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.Get(
		c.Request.Context(),
		&serviceports.GetBankReceiptWorkItemRequest{
			WorkItemID: id,
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
func (h *Handler) assign(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("workItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		AssignedToUserID pulid.ID `json:"assignedToUserId"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.Assign(
		c.Request.Context(),
		&serviceports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       id,
			AssignedToUserID: body.AssignedToUserID,
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
	c.JSON(http.StatusOK, item)
}
func (h *Handler) startReview(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("workItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.StartReview(
		c.Request.Context(),
		&serviceports.GetBankReceiptWorkItemRequest{
			WorkItemID: id,
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
	c.JSON(http.StatusOK, item)
}
func (h *Handler) resolve(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("workItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		ResolutionType string `json:"resolutionType"`
		ResolutionNote string `json:"resolutionNote"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.Resolve(
		c.Request.Context(),
		&serviceports.ResolveBankReceiptWorkItemRequest{
			WorkItemID:     id,
			ResolutionType: bankreceiptworkitem.ResolutionType(body.ResolutionType),
			ResolutionNote: body.ResolutionNote,
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
	c.JSON(http.StatusOK, item)
}
func (h *Handler) dismiss(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("workItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		ResolutionNote string `json:"resolutionNote"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.Dismiss(
		c.Request.Context(),
		&serviceports.DismissBankReceiptWorkItemRequest{
			WorkItemID:     id,
			ResolutionNote: body.ResolutionNote,
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
	c.JSON(http.StatusOK, item)
}
