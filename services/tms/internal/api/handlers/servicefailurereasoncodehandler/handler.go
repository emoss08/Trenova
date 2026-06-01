package servicefailurereasoncodehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.ServiceFailureReasonCodeService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.ServiceFailureReasonCodeService
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
	api := rg.Group("/service-failure-reason-codes")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpCreate),
		h.create,
	)
	api.GET(
		"/select-options/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpRead),
		h.selectOptions,
	)
	api.POST(
		"/reorder/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpUpdate),
		h.reorder,
	)
	api.GET(
		"/:reasonCodeID/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/:reasonCodeID/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:reasonCodeID/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/:reasonCodeID/archive/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpArchive),
		h.archive,
	)
	api.POST(
		"/:reasonCodeID/activate/",
		h.pm.RequirePermission(permission.ResourceServiceFailureReasonCode.String(), permission.OpArchive),
		h.activate,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*servicefailure.ReasonCode], error) {
		return h.service.List(c.Request.Context(), &repositories.ListServiceFailureReasonCodesRequest{
			Filter: req,
		})
	})
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reasonCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetServiceFailureReasonCodeByIDRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entity := new(servicefailure.ReasonCode)
	authctx.AddContextToRequest(authCtx, entity)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	created, err := h.service.Create(c.Request.Context(), entity, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reasonCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity := new(servicefailure.ReasonCode)
	entity.ID = id
	authctx.AddContextToRequest(authCtx, entity)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.Update(c.Request.Context(), entity, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reasonCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Get(c.Request.Context(), repositories.GetServiceFailureReasonCodeByIDRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.Update(c.Request.Context(), entity, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) archive(c *gin.Context) {
	h.toggleActive(c, false)
}

func (h *Handler) activate(c *gin.Context) {
	h.toggleActive(c, true)
}

func (h *Handler) toggleActive(c *gin.Context, active bool) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("reasonCodeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo := pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID}
	if active {
		entity, activateErr := h.service.Activate(c.Request.Context(), id, tenantInfo, actorutil.FromAuthContext(authCtx))
		if activateErr != nil {
			h.eh.HandleError(c, activateErr)
			return
		}
		c.JSON(http.StatusOK, entity)
		return
	}
	entity, archiveErr := h.service.Archive(c.Request.Context(), id, tenantInfo, actorutil.FromAuthContext(authCtx))
	if archiveErr != nil {
		h.eh.HandleError(c, archiveErr)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) reorder(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(repositories.ReorderServiceFailureReasonCodesRequest)
	req.TenantInfo = pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID}
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entities, err := h.service.Reorder(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entities)
}

func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	appliesTo := servicefailure.ReasonCodeAppliesTo(c.Query("appliesTo"))

	pagination.SelectOptions(c, req, h.eh, func() (*pagination.ListResult[*servicefailure.ReasonCode], error) {
		return h.service.SelectOptions(
			c.Request.Context(),
			&repositories.ServiceFailureReasonCodeSelectOptionsRequest{
				SelectQueryRequest: req,
				AppliesTo:          appliesTo,
			},
		)
	})
}
