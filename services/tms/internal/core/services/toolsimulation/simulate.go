// Package toolsimulation previews a write instead of making it, for an
// agent in simulation. A tool that knows how to preview itself does; any
// other is described by its name and parameters, and the preview says so.
package toolsimulation

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/uptrace/bun"
)

// FromBaseline is the simulation a baseline already took in its read-only
// snapshot. A preview that failed there is never run again outside it: the
// failure is the reason no more can be said, and a write the snapshot
// refused must not get a second chance to land.
func FromBaseline(
	toolName string,
	params map[string]any,
	baseline *serviceports.ProposalBaselineResult,
) *agent.ToolSimulation {
	if baseline == nil {
		return Describe(toolName, params)
	}
	if baseline.Preview != nil {
		return baseline.Preview.Bounded().Simulation()
	}
	if baseline.PreviewErr != nil {
		return failedPreview(toolName, params, baseline.PreviewErr)
	}

	return Describe(toolName, params)
}

// InSnapshot is Simulate inside one read-only, repeatable-read transaction,
// so a write the preview attempted fails there instead of landing. Without a
// database it is Simulate alone.
func InSnapshot(
	ctx context.Context,
	db ports.DBConnection,
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
) *agent.ToolSimulation {
	if db == nil {
		return Simulate(ctx, tool, params)
	}

	var simulation *agent.ToolSimulation
	err := db.WithTx(ctx, ports.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	}, func(txCtx context.Context, _ bun.Tx) error {
		simulation = Simulate(txCtx, tool, params)

		return nil
	})
	if simulation != nil {
		return simulation
	}
	if err != nil {
		return failedPreview(tool.Name(), params.Params, err)
	}

	return Describe(tool.Name(), params.Params)
}

// Simulate returns what the tool would change, for a caller with no
// snapshot of its own. The context is marked read-only, so work the
// preview hands off outside its transaction is skipped. A tool's own preview
// that fails is not a failure of the simulation: the fallback description
// still tells a person what was asked for, with the error as the reason no
// more could be said.
func Simulate(
	ctx context.Context,
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
) *agent.ToolSimulation {
	previewer, ok := tool.(serviceports.ToolPreviewer)
	if !ok {
		return Describe(tool.Name(), params.Params)
	}

	preview, err := previewer.Preview(ports.WithReadOnly(ctx), *params)
	if err != nil {
		return failedPreview(tool.Name(), params.Params, err)
	}
	if preview == nil {
		return Describe(tool.Name(), params.Params)
	}

	return preview.Bounded().Simulation()
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
