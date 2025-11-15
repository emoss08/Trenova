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

type WorkflowHandlerParams struct {
	fx.In

	Service      *workflowservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type WorkflowHandler struct {
	service      *workflowservice.Service
	errorHandler *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func NewWorkflowHandler(p WorkflowHandlerParams) *WorkflowHandler {
	return &WorkflowHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *WorkflowHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workflows/")
	api.GET("", h.pm.RequirePermission(permission.ResourceWorkflow, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceWorkflow, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceWorkflow, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceWorkflow, "update"), h.update)
	api.DELETE(":id/", h.pm.RequirePermission(permission.ResourceWorkflow, "delete"), h.delete)

	// Version management
	api.GET(
		":id/versions/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "read"),
		h.listVersions,
	)
	api.GET(
		":id/versions/:versionId/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "read"),
		h.getVersion,
	)
	api.POST(
		":id/versions/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.createVersion,
	)
	api.POST(
		":id/versions/:versionId/publish/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.publishVersion,
	)

	// Node and edge management
	api.PUT(
		":id/versions/:versionId/definition/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.saveDefinition,
	)

	// Status management
	api.POST(
		":id/activate/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.activate,
	)
	api.POST(
		":id/deactivate/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.deactivate,
	)
	api.POST(
		":id/archive/",
		h.pm.RequirePermission(permission.ResourceWorkflow, "update"),
		h.archive,
	)
}

func (h *WorkflowHandler) list(c *gin.Context) {
	pagination.Handle[*workflow.Workflow](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*workflow.Workflow], error) {
			return h.service.List(c.Request.Context(), &repositories.ListWorkflowRequest{
				Filter: opts,
			})
		})
}

func (h *WorkflowHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetWorkflowByIDRequest{
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

func (h *WorkflowHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(workflow.Workflow)
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

func (h *WorkflowHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(workflow.Workflow)
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

func (h *WorkflowHandler) delete(c *gin.Context) {
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

// Version Management

func (h *WorkflowHandler) listVersions(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	versions, err := h.service.GetVersions(
		c.Request.Context(),
		id,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

func (h *WorkflowHandler) getVersion(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	versionID, err := pulid.MustParse(c.Param("versionId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	version, err := h.service.GetVersion(
		c.Request.Context(),
		versionID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

type CreateVersionRequest struct {
	VersionName        string `json:"versionName"`
	Changelog          string `json:"changelog"`
	WorkflowDefinition any    `json:"workflowDefinition"`
}

func (h *WorkflowHandler) createVersion(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(CreateVersionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	version, err := h.service.CreateVersion(
		c.Request.Context(),
		id,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		authCtx.UserID,
		req.VersionName,
		req.Changelog,
		req.WorkflowDefinition,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, version)
}

func (h *WorkflowHandler) publishVersion(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	versionID, err := pulid.MustParse(c.Param("versionId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.PublishVersion(
		c.Request.Context(),
		id,
		versionID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		authCtx.UserID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version published successfully"})
}

// Node and Edge Management

type SaveDefinitionRequest struct {
	Nodes []*workflow.WorkflowNode `json:"nodes"`
	Edges []*workflow.WorkflowEdge `json:"edges"`
}

func (h *WorkflowHandler) saveDefinition(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	workflowID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	versionID, err := pulid.MustParse(c.Param("versionId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(SaveDefinitionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.SaveWorkflowDefinition(
		c.Request.Context(),
		&workflowservice.SaveWorkflowDefinitionRequest{
			WorkflowID: workflowID,
			VersionID:  versionID,
			OrgID:      authCtx.OrganizationID,
			BuID:       authCtx.BusinessUnitID,
			UserID:     authCtx.UserID,
			Nodes:      req.Nodes,
			Edges:      req.Edges,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow definition saved successfully"})
}

// Status Management

func (h *WorkflowHandler) activate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Activate(
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

	c.JSON(http.StatusOK, gin.H{"message": "Workflow activated successfully"})
}

func (h *WorkflowHandler) deactivate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Deactivate(
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

	c.JSON(http.StatusOK, gin.H{"message": "Workflow deactivated successfully"})
}

func (h *WorkflowHandler) archive(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Archive(
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

	c.JSON(http.StatusOK, gin.H{"message": "Workflow archived successfully"})
}
