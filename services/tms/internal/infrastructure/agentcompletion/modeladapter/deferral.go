package modeladapter

import "slices"

func deferredTools(tools []ToolSpec) map[string]struct{} {
	deferred := make(map[string]struct{}, len(tools))
	for idx := range tools {
		if tools[idx].Deferred {
			deferred[tools[idx].Name] = struct{}{}
		}
	}

	return deferred
}

func declaresLoadedTool(tools []ToolSpec) bool {
	return slices.ContainsFunc(tools, func(tool ToolSpec) bool { return !tool.Deferred })
}

func loadedTools(tools []ToolSpec, messages []Message) []ToolSpec {
	deferred := deferredTools(tools)
	if len(deferred) == 0 {
		return tools
	}
	loaded := make(map[string]struct{}, len(deferred))
	for idx := range messages {
		for _, name := range messages[idx].ToolReferences {
			loaded[name] = struct{}{}
		}
	}

	out := make([]ToolSpec, 0, len(tools)-len(deferred)+len(loaded))
	for _, tool := range tools {
		if _, isDeferred := deferred[tool.Name]; isDeferred {
			if _, isLoaded := loaded[tool.Name]; !isLoaded {
				continue
			}
		}
		tool.Deferred = false
		out = append(out, tool)
	}

	return out
}

func referencing(messages []Message, deferred map[string]struct{}) []Message {
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
