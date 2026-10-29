package openaiadapter

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

const toolChoiceNone = "none"

func toolChoiceWire(req *modeladapter.Request) string {
	if req.ToolChoiceNone && len(req.Tools) > 0 {
		return toolChoiceNone
	}

	return ""
}
