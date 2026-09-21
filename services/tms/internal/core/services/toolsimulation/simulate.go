// Package toolsimulation previews a write instead of making it, for an
// agent in simulation. A tool that knows how to preview itself does; any
// other is described by its name and parameters, and the preview says so.
package toolsimulation

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxParameterChars = 120

// Simulate returns what the tool would change. A tool's own preview that
// fails is not a failure of the simulation: the fallback description still
// tells a person what was asked for, with the error as the reason no more
// could be said.
func Simulate(
	ctx context.Context,
	tool serviceports.AgentTool,
	params serviceports.ToolExecuteParams,
) *agent.ToolSimulation {
	if simulator, ok := tool.(serviceports.ToolSimulator); ok {
		preview, err := simulator.Simulate(ctx, params)
		if err == nil && preview != nil {
			preview.Previewed = true

			return preview
		}
		if err != nil {
			fallback := Describe(tool.Name(), params.Params)
			fallback.Summary += " The preview could not be completed: " + err.Error()

			return fallback
		}
	}

	return Describe(tool.Name(), params.Params)
}

// Describe is the preview for a tool that has none of its own: what would
// be called, with what.
func Describe(toolName string, params map[string]any) *agent.ToolSimulation {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	changes := make([]agent.FieldChange, 0, len(keys))
	for _, key := range keys {
		changes = append(changes, agent.FieldChange{Field: key, To: describeValue(params[key])})
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would run %s with the parameters below. This tool has no preview of its own, "+
				"so what it would change is not shown.",
			toolName,
		),
		Changes: changes,
	}
}

func describeValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "nothing"
	case string:
		if strings.TrimSpace(v) == "" {
			return "nothing"
		}

		return stringutils.Ellipsize(v, maxParameterChars)
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprint(v)
	default:
		encoded, err := sonic.MarshalString(v)
		if err != nil {
			return fmt.Sprint(v)
		}

		return stringutils.Ellipsize(encoded, maxParameterChars)
	}
}
