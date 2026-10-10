package storedmileagehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/internal/modules/places/app/storedmileageservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/storedmileageports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *storedmileageservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *storedmileageservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/stored-mileages")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceStoredMileage.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:storedMileageID/",
		h.pm.RequirePermission(permission.ResourceStoredMileage.String(), permission.OpRead),
		h.get,
	)
	api.DELETE(
		"/:storedMileageID/",
		h.pm.RequirePermission(permission.ResourceStoredMileage.String(), permission.OpDelete),
		h.delete,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*storedmileage.StoredMileage], error) {
			return h.service.List(
				c.Request.Context(),
				&storedmileageports.ListStoredMileageRequest{Filter: req},
			)
		},
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("storedMileageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.Get(c.Request.Context(), storedmileageports.GetStoredMileageByIDRequest{
		ID:         id,
		TenantInfo: pagination.FromAuth(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("storedMileageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.Deactivate(c.Request.Context(), storedmileageports.DeleteStoredMileageRequest{
		ID:         id,
		TenantInfo: pagination.FromAuthAsUser(authCtx),
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
