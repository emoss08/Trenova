package modeladaptertest

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

const (
	WorldNotice = "[Notice from the system, not from the person: S-1 changed.]"
	SteerNotice = "[Notice from the system, not from the person: they added this.]"
)

func NoticeConversation() []modeladapter.Message {
	return []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Where is S-1?"},
		{Role: modeladapter.RoleAssistant, ToolCalls: []modeladapter.ToolCall{{
			ID:        "toolu_1",
			Name:      "get_shipment",
			Arguments: map[string]any{"number": "S-1"},
		}}},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "toolu_1",
			ToolName:   "get_shipment",
			Content:    `{"s":"X"}`,
		},
		{Role: modeladapter.RoleUser, Content: "Only the late ones."},
		{Role: modeladapter.RoleSystem, Content: SteerNotice},
		{Role: modeladapter.RoleSystem, Content: WorldNotice},
	}
}
