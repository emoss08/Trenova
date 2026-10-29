package modeladapter

func DeferredTools(tools []ToolSpec) map[string]struct{} {
	deferred := make(map[string]struct{}, len(tools))
	for idx := range tools {
		if tools[idx].Deferred {
			deferred[tools[idx].Name] = struct{}{}
		}
	}

	return deferred
}

func LoadedTools(tools []ToolSpec, messages []Message) []ToolSpec {
	deferred := DeferredTools(tools)
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
