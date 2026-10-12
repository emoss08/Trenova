package completionrouter

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
)

var _ aicorrectionports.StructuredPromptRenderer = PromptRenderer{}

type PromptRenderer struct{}

func NewPromptRenderer() aicorrectionports.StructuredPromptRenderer { return PromptRenderer{} }

func (PromptRenderer) RenderStructuredPrompt(
	req *agentports.StructuredCompletionRequest,
	mode aiprovider.StructuredOutputMode,
) aicorrectionports.RenderedPrompt {
	run := structuredRun(req)
	sampling := modeladapter.SamplingForTask(run.Task)

	return aicorrectionports.RenderedPrompt{
		System:      structuredSystem(run, mode),
		User:        run.UserContent,
		Temperature: sampling.Temperature,
		TopP:        sampling.TopP,
	}
}

func structuredRun(req *agentports.StructuredCompletionRequest) *runRequest {
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
		HedgeAfter:          req.HedgeAfter,
		Reasoning:           req.Reasoning,
		Attribution:         req.Attribution,
	}
}

func structuredSystem(run *runRequest, mode aiprovider.StructuredOutputMode) string {
	return modeladapter.WithSchemaInstruction(run.System, run.Schema, mode)
}
