package completionrouter

import (
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

var _ serviceports.StructuredPromptRenderer = PromptRenderer{}

type PromptRenderer struct{}

func NewPromptRenderer() serviceports.StructuredPromptRenderer { return PromptRenderer{} }

func (PromptRenderer) RenderStructuredPrompt(
	req *serviceports.StructuredCompletionRequest,
	mode aiprovider.StructuredOutputMode,
) serviceports.RenderedPrompt {
	run := structuredRun(req)
	sampling := modeladapter.SamplingForTask(run.Task)

	return serviceports.RenderedPrompt{
		System:      structuredSystem(run, mode),
		User:        run.UserContent,
		Temperature: sampling.Temperature,
		TopP:        sampling.TopP,
	}
}

func structuredRun(req *serviceports.StructuredCompletionRequest) *runRequest {
	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}

	return &runRequest{
		TenantInfo:          req.TenantInfo,
		Task:                task,
		System:              req.System,
		UserContent:         modeladapter.BuildContextText(req.Context),
		Schema:              req.OutputSchema,
		SchemaName:          req.SchemaName,
		MaxTokens:           req.MaxTokens,
		PreferredProviderID: req.PreferredProviderID,
		RequireProvider:     req.RequireProvider,
		Attribution:         req.Attribution,
	}
}

func structuredSystem(run *runRequest, mode aiprovider.StructuredOutputMode) string {
	return modeladapter.WithSchemaInstruction(run.System, run.Schema, mode)
}
