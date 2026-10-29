package toolaccess

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (s *Access) ToolEffect(name string) agenttooltrust.ToolEffect {
	if policy, ok := runtimetools.PolicyNamed(name); ok {
		return policy.Effect
	}

	return agenttoolports.EffectOf(s.ToolNamed(name))
}

func (s *Access) MarkToolEffects(messages []conversation.Message) {
	if s == nil {
		return
	}

	for idx := range messages {
		message := &messages[idx]
		if message.Role == conversation.RoleTool {
			message.ToolEffect = s.ToolEffect(message.ToolName)
		}
		if len(message.ToolCalls) > 0 {
			message.ToolCalls = s.CallsWithEffects(message.ToolCalls)
		}
	}
}

func (s *Access) CallsWithEffects(
	calls []conversation.ToolCallRecord,
) []conversation.ToolCallRecord {
	marked := slices.Clone(calls)
	for idx := range marked {
		marked[idx].Effect = s.ToolEffect(marked[idx].Name)
	}

	return marked
}
