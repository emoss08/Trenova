package journalreversalhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/journalreversalservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *journalreversalservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *journalreversalservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/journal-reversals")
	api.GET("/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpRead), h.list)
	api.GET("/:reversalID/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpCreate), h.create)
	api.POST("/:reversalID/approve/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpApprove), h.approve)
	api.POST("/:reversalID/post/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpApprove), h.post)
	api.POST("/:reversalID/reject/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpReject), h.reject)
	api.POST("/:reversalID/cancel/", h.pm.RequirePermission(permission.ResourceJournalReversal.String(), permission.OpCancel), h.cancel)
}

func (h *Handler) list(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	query := pagination.NewQueryOptions(c, auth)
	pagination.List(c, query, h.eh, func() (*pagination.ListResult[*journalreversal.Reversal], error) {
		return h.service.List(c.Request.Context(), &repositories.ListJournalReversalsRequest{Filter: query})
	})
}
func (h *Handler) get(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reversalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Get(c.Request.Context(), &serviceports.GetJournalReversalRequest{ReversalID: id, TenantInfo: pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
func (h *Handler) create(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := new(serviceports.CreateJournalReversalRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}
	entity, err := h.service.Create(c.Request.Context(), req, actorutil.FromAuthContext(auth))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, entity)
}
func (h *Handler) approve(c *gin.Context) {
	h.transitionNoBody(c, func(ctx *gin.Context, auth *authctx.AuthContext, id pulid.ID) (*journalreversal.Reversal, error) {
		return h.service.Approve(ctx.Request.Context(), &serviceports.GetJournalReversalRequest{ReversalID: id, TenantInfo: pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}}, actorutil.FromAuthContext(auth))
	})
}
func (h *Handler) post(c *gin.Context) {
	h.transitionNoBody(c, func(ctx *gin.Context, auth *authctx.AuthContext, id pulid.ID) (*journalreversal.Reversal, error) {
		return h.service.Post(ctx.Request.Context(), &serviceports.GetJournalReversalRequest{ReversalID: id, TenantInfo: pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}}, actorutil.FromAuthContext(auth))
	})
}
func (h *Handler) reject(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reversalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Reject(c.Request.Context(), &serviceports.RejectJournalReversalRequest{ReversalID: id, Reason: body.Reason, TenantInfo: pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}}, actorutil.FromAuthContext(auth))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
func (h *Handler) cancel(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reversalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Cancel(c.Request.Context(), &serviceports.CancelJournalReversalRequest{ReversalID: id, Reason: body.Reason, TenantInfo: pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}}, actorutil.FromAuthContext(auth))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
func (h *Handler) transitionNoBody(c *gin.Context, fn func(*gin.Context, *authctx.AuthContext, pulid.ID) (*journalreversal.Reversal, error)) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reversalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := fn(c, auth, id)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
