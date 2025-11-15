package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	workflowservice "github.com/emoss08/trenova/internal/core/services/workflowservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type WorkflowTemplateHandlerParams struct {
	fx.In

	Service      *workflowservice.TemplateService
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type WorkflowTemplateHandler struct {
	service      *workflowservice.TemplateService
	errorHandler *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func NewWorkflowTemplateHandler(p WorkflowTemplateHandlerParams) *WorkflowTemplateHandler {
	return &WorkflowTemplateHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *WorkflowTemplateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workflow-templates/")
	api.GET("", h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "read"), h.list)
	api.GET(
		"system/",
		h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "read"),
		h.listSystem,
	)
	api.GET(
		"public/",
		h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "read"),
		h.listPublic,
	)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "update"), h.update)
	api.DELETE(
		":id/",
		h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "delete"),
		h.delete,
	)
	api.POST(
		":id/use/",
		h.pm.RequirePermission(permission.ResourceWorkflowTemplate, "read"),
		h.useTemplate,
	)
}

func (h *WorkflowTemplateHandler) list(c *gin.Context) {
	// Optional category filter
	var category *string
	if cat := c.Query("category"); cat != "" {
		category = &cat
	}

	pagination.Handle[*workflow.WorkflowTemplate](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*workflow.WorkflowTemplate], error) {
			return h.service.List(c.Request.Context(), &repositories.ListWorkflowTemplateRequest{
				Filter:   opts,
				Category: category,
			})
		})
}

func (h *WorkflowTemplateHandler) listSystem(c *gin.Context) {
	templates, err := h.service.GetSystemTemplates(c.Request.Context())
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, templates)
}

func (h *WorkflowTemplateHandler) listPublic(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	templates, err := h.service.GetPublicTemplates(
		c.Request.Context(),
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, templates)
}

func (h *WorkflowTemplateHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetWorkflowTemplateByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *WorkflowTemplateHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(workflow.WorkflowTemplate)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *WorkflowTemplateHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(workflow.WorkflowTemplate)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *WorkflowTemplateHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Delete(
		c.Request.Context(),
		id,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		authCtx.UserID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *WorkflowTemplateHandler) useTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	template, err := h.service.UseTemplate(
		c.Request.Context(),
		id,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		authCtx.UserID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}
