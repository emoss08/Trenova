package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type VariableHandlerParams struct {
	fx.In

	Service      services.VariableService
	ErrorHandler *helpers.ErrorHandler
}

type VariableHandler struct {
	service services.VariableService
	eh      *helpers.ErrorHandler
}

func NewVariableHandler(p VariableHandlerParams) *VariableHandler {
	return &VariableHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *VariableHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/variables/")
	api.GET("", h.list)
	api.POST("", h.create)
	api.GET(":id/", h.get)
	api.PUT(":id/", h.update)
	api.DELETE(":id/", h.delete)
	api.POST("validate/", h.validateQuery)
	api.POST("test/", h.testVariable)
	api.GET("context/:context/", h.getByContext)

	formats := rg.Group("/variable-formats/")
	formats.GET("", h.listFormats)
	formats.POST("", h.createFormat)
	formats.GET(":id/", h.getFormat)
	formats.PUT(":id/", h.updateFormat)
	formats.DELETE(":id/", h.deleteFormat)
	formats.POST("validate/", h.validateFormatSQL)
	formats.POST("test/", h.testFormat)
}

func (h *VariableHandler) list(c *gin.Context) {
	pagination.Handle[*variable.Variable](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*variable.Variable], error) {
			return h.service.List(c.Request.Context(), &repositories.ListVariableRequest{
				Filter:        opts,
				IncludeFormat: helpers.QueryBool(c, "includeFormat"),
			})
		})
}

func (h *VariableHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetVariableByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *VariableHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(variable.Variable)
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

func (h *VariableHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(variable.Variable)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)
	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *VariableHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.Delete(
		c.Request.Context(),
		repositories.GetVariableByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *VariableHandler) validateQuery(c *gin.Context) {
	var req services.ValidateQueryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.ValidateQuery(&req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *VariableHandler) testVariable(c *gin.Context) {
	var req services.TestVariableRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.TestVariable(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *VariableHandler) getByContext(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	contextStr := c.Param("context")
	vContext, err := variable.ParseContext(contextStr)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	variables, err := h.service.GetVariablesByContext(
		c.Request.Context(),
		repositories.GetVariablesByContextRequest{
			OrgID:   authCtx.OrganizationID,
			Context: vContext,
			Active:  true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, variables)
}

func (h *VariableHandler) listFormats(c *gin.Context) {
	pagination.Handle[*variable.VariableFormat](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*variable.VariableFormat], error) {
			return h.service.ListFormats(
				c.Request.Context(),
				&repositories.ListVariableFormatRequest{
					Filter: opts,
				},
			)
		})
}

func (h *VariableHandler) getFormat(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetFormat(
		c.Request.Context(),
		repositories.GetVariableFormatByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *VariableHandler) createFormat(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(variable.VariableFormat)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.CreateFormat(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *VariableHandler) updateFormat(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(variable.VariableFormat)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)
	entity, err = h.service.UpdateFormat(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *VariableHandler) deleteFormat(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteFormat(
		c.Request.Context(),
		repositories.GetVariableFormatByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *VariableHandler) validateFormatSQL(c *gin.Context) {
	var req services.ValidateFormatSQLRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.ValidateFormatSQL(&req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *VariableHandler) testFormat(c *gin.Context) {
	var req services.TestFormatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	resp, err := h.service.TestFormat(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
