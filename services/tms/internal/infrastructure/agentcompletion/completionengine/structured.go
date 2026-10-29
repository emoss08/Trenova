package completionengine

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

func StructuredRun(req *llmports.StructuredCompletionRequest) *RunRequest {
	task := req.Task
	if task == "" {
		task = aitask.TaskGeneral
	}

	return &RunRequest{
		TenantInfo:          req.TenantInfo,
		Task:                task,
		System:              req.System,
		UserContent:         modeladapter.BuildContextText(req.Context),
		Schema:              req.OutputSchema,
		SchemaName:          req.SchemaName,
		MaxTokens:           req.MaxTokens,
		PreferredProviderID: req.PreferredProviderID,
		RequireProvider:     req.RequireProvider,
		HedgeAfter:          req.HedgeAfter,
		Reasoning:           req.Reasoning,
		Attribution:         req.Attribution,
	}
}

func StructuredSystem(run *RunRequest, mode aiprovider.StructuredOutputMode) string {
	return modeladapter.WithSchemaInstruction(run.System, run.Schema, mode)
}
