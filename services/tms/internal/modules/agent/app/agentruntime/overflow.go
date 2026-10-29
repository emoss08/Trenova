package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aimodel"
)

// window is the context window of the model answering the turn, or zero
// before any has answered. The first call carries no results of the turn's
// own, so there is nothing to squeeze before it.
func (t *Turn) window() int {
	if t.result == nil || t.result.Model == "" {
		return 0
	}

	return aimodel.WindowOr(t.result.ContextWindow, t.result.Model)
}
