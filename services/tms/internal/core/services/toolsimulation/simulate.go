// Package toolsimulation previews a write instead of making it, for an
// agent in simulation. A tool that knows how to preview itself does; any
// other is described by its name and parameters, and the preview says so.
package toolsimulation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

// Simulate returns what the tool would change. A tool's own preview that
// fails is not a failure of the simulation: the fallback description still
// tells a person what was asked for, with the error as the reason no more
// could be said. A tool that previews record by record is read that way; one
// that only simulates is read as it simulates.
func Simulate(
	ctx context.Context,
	tool serviceports.AgentTool,
	params serviceports.ToolExecuteParams,
) *agent.ToolSimulation {
	if previewer, ok := tool.(serviceports.ToolPreviewer); ok {
		preview, err := previewer.Preview(ctx, params)
		if err == nil && preview != nil {
			return preview.Bounded().Simulation()
		}
		if err != nil {
			return failedPreview(tool.Name(), params.Params, err)
		}
	}

	if simulator, ok := tool.(serviceports.ToolSimulator); ok {
		preview, err := simulator.Simulate(ctx, params)
		if err == nil && preview != nil {
			preview.Previewed = true

			return preview
		}
		if err != nil {
			return failedPreview(tool.Name(), params.Params, err)
		}
	}

	return Describe(tool.Name(), params.Params)
}

// Describe is the preview for a tool that has none of its own: what would
// be called, with what.
func Describe(toolName string, params map[string]any) *agent.ToolSimulation {
	simulation := toolpreview.Describe(toolName, params).Simulation()
	simulation.Previewed = false

	return simulation
}

func failedPreview(toolName string, params map[string]any, err error) *agent.ToolSimulation {
	fallback := Describe(toolName, params)
	fallback.Summary += " The preview could not be completed: " + err.Error()

	return fallback
}
