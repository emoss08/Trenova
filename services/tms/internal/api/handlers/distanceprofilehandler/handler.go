package distanceprofilehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/distanceprofileservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *distanceprofileservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *distanceprofileservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/distance-profiles")
	api.GET("/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpRead), h.list)
	api.GET("/select-options/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpRead), h.selectOptions)
	api.POST("/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpCreate), h.create)
	api.GET("/:distanceProfileID/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpRead), h.get)
	api.PUT("/:distanceProfileID/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpUpdate), h.update)
	api.PATCH("/:distanceProfileID/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpUpdate), h.patch)
	api.DELETE("/:distanceProfileID/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpDelete), h.delete)
	api.POST("/:distanceProfileID/set-default/", h.pm.RequirePermission(permission.ResourceDistanceProfile.String(), permission.OpUpdate), h.setDefault)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
		return h.service.List(c.Request.Context(), &repositories.ListDistanceProfileRequest{Filter: req})
	})
}

func (h *Handler) selectOptions(c *gin.Context) {
	h.list(c)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("distanceProfileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Get(c.Request.Context(), repositories.GetDistanceProfileByIDRequest{
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
	entity := new(distanceprofile.DistanceProfile)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	authctx.AddContextToRequest(authCtx, entity)
	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("distanceProfileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity := new(distanceprofile.DistanceProfile)
	entity.ID = id
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	authctx.AddContextToRequest(authCtx, entity)
	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("distanceProfileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	existing, err := h.service.Get(c.Request.Context(), repositories.GetDistanceProfileByIDRequest{
		ID: id,
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
	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("distanceProfileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.Delete(c.Request.Context(), repositories.DeleteDistanceProfileRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) setDefault(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("distanceProfileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.SetDefault(c.Request.Context(), repositories.GetDistanceProfileByIDRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}
