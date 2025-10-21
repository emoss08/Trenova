package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/user"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type UserHandlerParams struct {
	fx.In

	Service      *user.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type UserHandler struct {
	service *user.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewUserHandler(p UserHandlerParams) *UserHandler {
	return &UserHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/users/")
	api.GET("", h.pm.RequirePermission(permission.ResourceUser, "read"), h.list)
	api.GET("me/", h.me)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceUser, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceUser, "create"), h.create)
	api.POST("change-password/", h.changePassword)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceUser, "update"), h.update)

	selectOptions := api.Group("/select-options/")
	selectOptions.GET("", h.selectOptions)
	selectOptions.GET(":id/", h.getOption)
}

func (h *UserHandler) list(c *gin.Context) {
	pagination.Handle[*tenant.User](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tenant.User], error) {
			return h.service.List(c.Request.Context(), &repositories.ListUserRequest{
				Filter:       opts,
				IncludeRoles: helpers.QueryBool(c, "includeRoles"),
			})
		})
}

func (h *UserHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		UserID:       userID,
		BuID:         authCtx.BusinessUnitID,
		OrgID:        authCtx.OrganizationID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *UserHandler) getOption(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetOption(c.Request.Context(), repositories.GetUserByIDRequest{
		UserID: userID,
		BuID:   authCtx.BusinessUnitID,
		OrgID:  authCtx.OrganizationID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *UserHandler) selectOptions(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	options, err := h.service.SelectOptions(
		c.Request.Context(),
		repositories.UserSelectOptionsRequest{
			SelectQueryOptions: &pagination.SelectQueryOptions{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				Limit:  helpers.QueryInt(c, "limit", 20),
				Offset: helpers.QueryInt(c, "offset", 0),
				Query:  helpers.QueryString(c, "query"),
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": options})
}

func (h *UserHandler) me(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	usr, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		UserID:       authCtx.UserID,
		BuID:         authCtx.BusinessUnitID,
		OrgID:        authCtx.OrganizationID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, usr)
}

func (h *UserHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(tenant.User)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *UserHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tenant.User)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = userID
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *UserHandler) changePassword(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.ChangePasswordRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	entity, err := h.service.ChangePassword(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
