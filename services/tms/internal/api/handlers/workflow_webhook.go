package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	workflowservice "github.com/emoss08/trenova/internal/core/services/workflowservice"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type WorkflowWebhookHandlerParams struct {
	fx.In

	TriggerService *workflowservice.TriggerService
	ErrorHandler   *helpers.ErrorHandler
	Logger         *zap.Logger
}

type WorkflowWebhookHandler struct {
	triggerService *workflowservice.TriggerService
	errorHandler   *helpers.ErrorHandler
	logger         *zap.Logger
}

func NewWorkflowWebhookHandler(p WorkflowWebhookHandlerParams) *WorkflowWebhookHandler {
	return &WorkflowWebhookHandler{
		triggerService: p.TriggerService,
		errorHandler:   p.ErrorHandler,
		logger:         p.Logger.Named("workflow-webhook-handler"),
	}
}

func (h *WorkflowWebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Public webhook endpoint (no auth required - uses workflow-specific auth)
	api := rg.Group("/workflow-webhooks/")
	api.POST(":workflowId/", h.trigger)
}

type WebhookTriggerRequest struct {
	Data      map[string]any `json:"data"`
	AuthToken string         `json:"authToken,omitempty" header:"X-Workflow-Auth"`
}

func (h *WorkflowWebhookHandler) trigger(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	workflowID, err := pulid.MustParse(c.Param("workflowId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	var req WebhookTriggerRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	// Try to get auth token from header if not in body
	if req.AuthToken == "" {
		req.AuthToken = c.GetHeader("X-Workflow-Auth")
	}

	// TODO: Validate webhook auth token against workflow configuration
	// For now, we'll use the authenticated user's context
	// In production, you'd want to verify the auth token matches the workflow's webhook config

	execution, err := h.triggerService.TriggerWebhookExecution(
		c.Request.Context(),
		workflowID,
		req.Data,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		authCtx.UserID,
	)
	if err != nil {
		h.logger.Error("Failed to trigger workflow via webhook",
			zap.String("workflowId", workflowID.String()),
			zap.Error(err),
		)
		h.errorHandler.HandleError(c, err)
		return
	}

	h.logger.Info("Workflow triggered via webhook",
		zap.String("workflowId", workflowID.String()),
		zap.String("executionId", execution.ID.String()),
	)

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Workflow triggered successfully",
		"executionId": execution.ID.String(),
		"status":      string(execution.Status),
	})
}
