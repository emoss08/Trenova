package openaiadapter

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

func answerRoom(call *modeladapter.Call, ceiling int) int {
	if !call.Reasons() {
		return ceiling
	}

	return max(ceiling, modeladapter.ReasoningAnswerFloor)
}
