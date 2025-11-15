package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	workflowservice "github.com/emoss08/trenova/internal/core/services/workflowservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type WorkflowExecutionHandlerParams struct {
	fx.In

	Service      *workflowservice.ExecutionService
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type WorkflowExecutionHandler struct {
	service      *workflowservice.ExecutionService
	errorHandler *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func NewWorkflowExecutionHandler(p WorkflowExecutionHandlerParams) *WorkflowExecutionHandler {
	return &WorkflowExecutionHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
		pm:           p.PM,
	}
}

func (h *WorkflowExecutionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workflow-executions/")
	api.GET("", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "read"), h.get)
	api.GET(":id/steps/", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "read"), h.getSteps)
	api.POST(":id/cancel/", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "update"), h.cancel)
	api.POST(":id/retry/", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "create"), h.retry)

	// Trigger workflow execution
	api.POST("trigger/:workflowId/", h.pm.RequirePermission(workflow.ResourceWorkflowExecution, "create"), h.trigger)
}

func (h *WorkflowExecutionHandler) list(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	// Optional filters
	var workflowID *pulid.ID
	if wfID := c.Query("workflowId"); wfID != "" {
		id, err := pulid.MustParse(wfID)
		if err == nil {
			workflowID = &id
		}
	}

	var status *workflow.ExecutionStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := workflow.ExecutionStatus(statusStr)
		status = &s
	}

	pagination.Handle[*workflow.WorkflowExecution](c, authCtx).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*workflow.WorkflowExecution], error) {
			return h.service.List(c.Request.Context(), &repositories.ListWorkflowExecutionRequest{
				Filter:     opts,
				WorkflowID: workflowID,
				Status:     status,
			})
		})
}

func (h *WorkflowExecutionHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetWorkflowExecutionByIDRequest{
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

func (h *WorkflowExecutionHandler) getSteps(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	steps, err := h.service.GetSteps(c.Request.Context(), id, authCtx.OrganizationID, authCtx.BusinessUnitID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, steps)
}

type TriggerWorkflowRequest struct {
	TriggerData map[string]any `json:"triggerData"`
}

func (h *WorkflowExecutionHandler) trigger(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	workflowID, err := pulid.MustParse(c.Param("workflowId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(TriggerWorkflowRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	execution, err := h.service.TriggerWorkflow(c.Request.Context(), &workflowservice.TriggerWorkflowRequest{
		WorkflowID:  workflowID,
		OrgID:       authCtx.OrganizationID,
		BuID:        authCtx.BusinessUnitID,
		UserID:      authCtx.UserID,
		TriggerData: req.TriggerData,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, execution)
}

func (h *WorkflowExecutionHandler) cancel(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.CancelExecution(c.Request.Context(), id, authCtx.OrganizationID, authCtx.BusinessUnitID, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Execution canceled successfully"})
}

func (h *WorkflowExecutionHandler) retry(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	execution, err := h.service.RetryExecution(c.Request.Context(), id, authCtx.OrganizationID, authCtx.BusinessUnitID, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, execution)
}
