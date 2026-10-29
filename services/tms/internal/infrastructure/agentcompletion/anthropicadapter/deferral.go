package anthropicadapter

import (
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

func declaresLoadedTool(tools []modeladapter.ToolSpec) bool {
	return slices.ContainsFunc(
		tools,
		func(tool modeladapter.ToolSpec) bool { return !tool.Deferred },
	)
}

func referencing(
	messages []modeladapter.Message,
	deferred map[string]struct{},
) []modeladapter.Message {
	out := slices.Clone(messages)
	for idx := range out {
		refs := out[idx].ToolReferences
		if len(refs) == 0 {
			continue
		}
		kept := make([]string, 0, len(refs))
		for _, name := range refs {
			if _, ok := deferred[name]; ok && !slices.Contains(kept, name) {
				kept = append(kept, name)
			}
		}
		out[idx].ToolReferences = kept
	}

	return out
}
