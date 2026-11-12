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

// OpenedTools declares in full the deferred tools a person's message opened
// its turn with, for a protocol that can only load a tool on a tool result,
// and takes the references off those messages.
func OpenedTools(tools []ToolSpec, messages []Message) ([]ToolSpec, []Message) {
	opened := make(map[string]struct{}, 8)
	for idx := range messages {
		if messages[idx].Role != RoleUser {
			continue
		}
		for _, name := range messages[idx].ToolReferences {
			opened[name] = struct{}{}
		}
	}
	if len(opened) == 0 {
		return tools, messages
	}

	declared := make([]ToolSpec, len(tools))
	for idx, tool := range tools {
		if _, ok := opened[tool.Name]; ok {
			tool.Deferred = false
		}
		declared[idx] = tool
	}
	asked := make([]Message, len(messages))
	for idx, message := range messages {
		if message.Role == RoleUser {
			message.ToolReferences = nil
		}
		asked[idx] = message
	}

	return declared, asked
}
