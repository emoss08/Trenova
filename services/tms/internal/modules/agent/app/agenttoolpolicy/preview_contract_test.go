package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func TestEveryActionToolPreviewsWhatItWouldDo(t *testing.T) {
	t.Parallel()

	for _, tool := range buildRegistered(t).Actions {
		if tool.Policy().Kind != agent.ToolKindAction {
			continue
		}

		if _, previews := tool.(agentports.ToolPreviewer); !previews {
			t.Errorf(
				"%s is an action tool without a previewer; a person would approve it blind",
				tool.Name(),
			)
		}
	}
}
