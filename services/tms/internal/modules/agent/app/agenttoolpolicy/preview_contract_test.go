package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func TestEveryActionToolPreviewsWhatItWouldDo(t *testing.T) {
	t.Parallel()

	for _, tool := range buildRegistered(t).Actions {
		if tool.Policy().Kind != agenttooltrust.ToolKindAction {
			continue
		}

		if _, previews := tool.(agenttoolports.ToolPreviewer); !previews {
			t.Errorf(
				"%s is an action tool without a previewer; a person would approve it blind",
				tool.Name(),
			)
		}
	}
}
