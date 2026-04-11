package manualjournalhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/manualjournalservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *manualjournalservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *manualjournalservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/manual-journals")
	api.GET("/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpRead), h.list)
	api.GET("/:requestID/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpRead), h.get)
	api.POST("/drafts/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpCreate), h.createDraft)
	api.PUT("/drafts/:requestID/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpUpdate), h.updateDraft)
	api.POST("/:requestID/submit/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpSubmit), h.submit)
	api.POST("/:requestID/approve/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpApprove), h.approve)
	api.POST("/:requestID/post/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpApprove), h.post)
	api.POST("/:requestID/reject/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpReject), h.reject)
	api.POST("/:requestID/cancel/", h.pm.RequirePermission(permission.ResourceManualJournal.String(), permission.OpCancel), h.cancel)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	query := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, query, h.eh, func() (*pagination.ListResult[*manualjournal.Request], error) {
		return h.service.List(c.Request.Context(), &repositories.ListManualJournalRequest{Filter: query})
	})
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	requestID, err := pulid.MustParse(c.Param("requestID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), &serviceports.GetManualJournalRequest{
		RequestID:  requestID,
		TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(serviceports.CreateManualJournalRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID}

	entity, err := h.service.CreateDraft(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *Handler) updateDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	requestID, err := pulid.MustParse(c.Param("requestID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(serviceports.UpdateManualJournalDraftRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.RequestID = requestID
	req.TenantInfo = pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID}

	entity, err := h.service.UpdateDraft(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) submit(c *gin.Context) {
	h.transitionNoBody(c, func(ctx *gin.Context, authCtx *authctx.AuthContext, requestID pulid.ID) (*manualjournal.Request, error) {
		return h.service.Submit(ctx.Request.Context(), &serviceports.GetManualJournalRequest{
			RequestID:  requestID,
			TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
		}, actorutil.FromAuthContext(authCtx))
	})
}

func (h *Handler) approve(c *gin.Context) {
	h.transitionNoBody(c, func(ctx *gin.Context, authCtx *authctx.AuthContext, requestID pulid.ID) (*manualjournal.Request, error) {
		return h.service.Approve(ctx.Request.Context(), &serviceports.GetManualJournalRequest{
			RequestID:  requestID,
			TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
		}, actorutil.FromAuthContext(authCtx))
	})
}

func (h *Handler) post(c *gin.Context) {
	h.transitionNoBody(c, func(ctx *gin.Context, authCtx *authctx.AuthContext, requestID pulid.ID) (*manualjournal.Request, error) {
		return h.service.Post(ctx.Request.Context(), &serviceports.GetManualJournalRequest{
			RequestID:  requestID,
			TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
		}, actorutil.FromAuthContext(authCtx))
	})
}

func (h *Handler) reject(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	requestID, err := pulid.MustParse(c.Param("requestID"))
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

	entity, err := h.service.Reject(c.Request.Context(), &serviceports.RejectManualJournalRequest{
		RequestID:  requestID,
		Reason:     body.Reason,
		TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) cancel(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	requestID, err := pulid.MustParse(c.Param("requestID"))
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

	entity, err := h.service.Cancel(c.Request.Context(), &serviceports.CancelManualJournalRequest{
		RequestID:  requestID,
		Reason:     body.Reason,
		TenantInfo: pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID, UserID: authCtx.UserID},
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) transitionNoBody(
	c *gin.Context,
	fn func(*gin.Context, *authctx.AuthContext, pulid.ID) (*manualjournal.Request, error),
) {
	authCtx := authctx.GetAuthContext(c)
	requestID, err := pulid.MustParse(c.Param("requestID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := fn(c, authCtx, requestID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
