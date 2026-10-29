package modeladaptertest

import (
	"strconv"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

// wideRequest carries tool schemas and replayed tool-call arguments with
// enough keys that an unordered encoding would differ between calls.
func WideRequest() *modeladapter.Request {
	properties := map[string]any{}
	arguments := map[string]any{}
	for i := range 24 {
		key := "field_" + strconv.Itoa(i)
		properties[key] = map[string]any{"type": "string", "description": "field " + key}
		arguments[key] = "value " + key
	}

	return &modeladapter.Request{
		System: "be brief",
		Messages: []modeladapter.Message{
			{Role: modeladapter.RoleUser, Content: "look these up"},
			{
				Role: modeladapter.RoleAssistant,
				ToolCalls: []modeladapter.ToolCall{
					{ID: "call_1", Name: "lookup_wide", Arguments: arguments},
				},
			},
			{Role: modeladapter.RoleTool, ToolCallID: "call_1", Content: "{}"},
			{Role: modeladapter.RoleUser, Content: "and again"},
		},
		Tools: []modeladapter.ToolSpec{{
			Name:        "lookup_wide",
			Description: "Look up many fields",
			Parameters:  map[string]any{"type": "object", "properties": properties},
		}},
	}
}
