package completionrouter

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

var _ aicorrectionports.StructuredPromptRenderer = PromptRenderer{}

type PromptRenderer struct{}

func NewPromptRenderer() aicorrectionports.StructuredPromptRenderer { return PromptRenderer{} }

func (PromptRenderer) RenderStructuredPrompt(
	req *llmports.StructuredCompletionRequest,
	mode aiprovider.StructuredOutputMode,
) aicorrectionports.RenderedPrompt {
	run := completionengine.StructuredRun(req)
	sampling := modeladapter.SamplingForTask(run.Task)

	return aicorrectionports.RenderedPrompt{
		System:      completionengine.StructuredSystem(run, mode),
		User:        run.UserContent,
		Temperature: sampling.Temperature,
		TopP:        sampling.TopP,
	}
}
