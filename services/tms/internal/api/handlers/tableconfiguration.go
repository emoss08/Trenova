package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	tcService "github.com/emoss08/trenova/internal/core/services/tableconfiguration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type TableConfigurationHandlerParams struct {
	fx.In

	Service      *tcService.Service
	ErrorHandler *helpers.ErrorHandler
}

type TableConfigurationHandler struct {
	service *tcService.Service
	eh      *helpers.ErrorHandler
}

func NewTableConfigurationHandler(p TableConfigurationHandlerParams) *TableConfigurationHandler {
	return &TableConfigurationHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *TableConfigurationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/table-configurations/")
	api.GET("", h.list)
	api.GET("me/:resource/", h.listUserConfigurations)
	api.GET(":resource/", h.getDefaultOrLatest)
	api.GET("public/:resource/", h.listPublicConfigurations)
	api.PUT(":id/", h.update)
	api.POST("", h.create)
	api.POST("share/", h.share)
	api.POST("copy/", h.copy)
	api.DELETE(":id/", h.delete)
}

func (h *TableConfigurationHandler) list(c *gin.Context) {
	var req repositories.TableConfigurationFilters
	pagination.Handle[*tableconfiguration.Configuration](
		c,
		context.GetAuthContext(c),
	).
		WithErrorHandler(h.eh).
		WithExtraParams(&req).
		Execute(func(c *gin.Context, _ *pagination.QueryOptions) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
			return h.service.List(c.Request.Context(), &req)
		})
}

func (h *TableConfigurationHandler) listUserConfigurations(c *gin.Context) {
	pagination.Handle[*tableconfiguration.Configuration](
		c,
		context.GetAuthContext(c),
	).
		WithErrorHandler(h.eh).
		ExecuteWithHandler(func(ctx *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
			resource := ctx.Param("resource")
			return h.service.ListUserConfigurations(
				ctx.Request.Context(),
				&repositories.ListUserConfigurationRequest{
					Resource: resource,
					Filter:   opts,
				},
			)
		})
}

func (h *TableConfigurationHandler) listPublicConfigurations(c *gin.Context) {
	pagination.Handle[*tableconfiguration.Configuration](
		c,
		context.GetAuthContext(c),
	).
		WithErrorHandler(h.eh).
		ExecuteWithHandler(func(ctx *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
			resource := ctx.Param("resource")
			return h.service.ListPublicConfigurations(
				ctx.Request.Context(),
				&repositories.TableConfigurationFilters{
					Resource: resource,
					Filter:   opts,
				},
			)
		})
}

func (h *TableConfigurationHandler) getDefaultOrLatest(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	resource := c.Param("resource")
	config, err := h.service.GetDefaultOrLatestConfiguration(c.Request.Context(), resource, authCtx)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *TableConfigurationHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	config := new(tableconfiguration.Configuration)
	if err := c.ShouldBindJSON(config); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, config)
	config, err := h.service.Create(c.Request.Context(), config)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, config)
}

func (h *TableConfigurationHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	configID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	config := new(tableconfiguration.Configuration)
	if err = c.ShouldBindJSON(config); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	config.ID = configID
	context.AddContextToRequest(authCtx, config)
	config, err = h.service.Update(c.Request.Context(), config)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *TableConfigurationHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	configID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeleteUserConfigurationRequest{
		ConfigID: configID,
		UserID:   authCtx.UserID,
		OrgID:    authCtx.OrganizationID,
		BuID:     authCtx.BusinessUnitID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TableConfigurationHandler) copy(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.CopyTableConfigurationRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	if err := h.service.Copy(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TableConfigurationHandler) share(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	share := new(tableconfiguration.ConfigurationShare)
	if err := c.ShouldBindJSON(share); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, share)
	if err := h.service.ShareConfiguration(c.Request.Context(), share, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
